// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tool

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"text/template"
	"unicode"
	"unicode/utf8"
)

var Name string
var Title string
var FormatErr = PrintErrExit

// A Command is an implementation of a go command
// like go build or go fix.
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(cmd *Command, w io.Writer, args []string) error

	// UsageLine is the one-line usage message.
	// The first word in the line is taken to be the command name.
	UsageLine string

	// Short is the short description shown in the 'go help' output.
	Short string

	// Long is the long message shown in the 'go help <this-command>' output.
	Long string

	// Flag is a set of flags specific to this command.
	Flag flag.FlagSet

	// CustomFlags indicates that the command will do its own
	// flag parsing.
	CustomFlags bool

	Hidden       bool
	ExtraArgsReq int
	ExtraArgsMax int
}

// Name returns the command's name: the first word in the usage line.
func (c *Command) Name() string {
	name := c.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "%s\n", strings.TrimSpace(c.Long))
	os.Exit(2)
}

// Runnable reports whether the command can be run; otherwise
// it is a documentation pseudo-command such as importpath.
func (c *Command) Runnable() bool {
	return c.Run != nil
}

// Commands lists the available commands and help topics.
// The order here is the order in which they are printed by 'go help'.
var Commands []*Command

var exitStatus = 0
var exitMu sync.Mutex

func setExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}

func Run() {
	flag.Usage = usage
	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	if args[0] == "help" {
		help(args[1:])
		return
	}

	for _, cmd := range Commands {
		if cmd.Name() == args[0] && cmd.Runnable() {
			cmd.Flag.Usage = func() { cmd.Usage() }
			if cmd.CustomFlags {
				args = args[1:]
			} else {
				cmd.Flag.Parse(args[1:])
				args = cmd.Flag.Args()
			}
			switch {
			case len(args) < cmd.ExtraArgsReq:
				fmt.Fprintf(os.Stderr, Name+": too few arguments provided\n")
				cmd.Usage()
			case cmd.ExtraArgsMax == 0 && cmd.ExtraArgsReq > 0:
			case len(args) > cmd.ExtraArgsMax:
				fmt.Fprintf(os.Stderr, Name+": too many arguments provided\n")
				cmd.Usage()
			}
			w := bufio.NewWriter(os.Stdout)
			err := cmd.Run(cmd, w, args)
			w.Flush()
			if err != nil {
				if FormatErr != nil {
					FormatErr(err)
				} else {
					fmt.Fprintln(os.Stderr, err)
				}
				setExitStatus(1)
			}
			exit()
			return
		}
	}

	fmt.Fprintf(os.Stderr, Name+": unknown subcommand %q\nRun '"+Name+" help' for usage.\n", args[0])
	setExitStatus(2)
	exit()
}

var usageTemplate = func() string {
	return Title + `.

Usage:

	` + Name + ` command [arguments]

The commands are:
{{range .}}{{if and .Runnable (not .Hidden)}}
	{{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "` + Name + ` help [command]" for more information about a command.

Additional help topics:
{{range .}}{{if not .Runnable}}
	{{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "` + Name + ` help [topic]" for more information about that topic.

`
}

var helpTemplate = func() string {
	return `{{if .Runnable}}usage: ` + Name + ` {{.UsageLine}}

{{end}}{{.Long | trim}}
`
}

// An errWriter wraps a writer, recording whether a write error occurred.
type errWriter struct {
	w   io.Writer
	err error
}

func (w *errWriter) Write(b []byte) (int, error) {
	n, err := w.w.Write(b)
	if err != nil {
		w.err = err
	}
	return n, err
}

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace, "capitalize": capitalize})
	template.Must(t.Parse(text))
	ew := &errWriter{w: w}
	err := t.Execute(ew, data)
	if ew.err != nil {
		// I/O error writing. Ignore write on closed pipe.
		if strings.Contains(ew.err.Error(), "pipe") {
			os.Exit(1)
		}
		fatalf("writing output: %v", ew.err)
	}
	if err != nil {
		panic(err)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}

func printUsage(w io.Writer) {
	bw := bufio.NewWriter(w)
	tmpl(bw, usageTemplate(), Commands)
	bw.Flush()
}

func usage() {
	printUsage(os.Stderr)
	os.Exit(2)
}

// help implements the 'help' command.
func help(args []string) {
	if len(args) == 0 {
		printUsage(os.Stdout)
		// not exit 2: succeeded at '<prog> help'.
		return
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: "+Name+" help command\n\nToo many arguments given.\n")
		os.Exit(2) // failed at '<prog> help'
	}

	arg := args[0]

	for _, cmd := range Commands {
		if cmd.Name() == arg {
			tmpl(os.Stdout, helpTemplate(), cmd)
			// not exit 2: succeeded at '<prog> help cmd'.
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown help topic %#q.  Run 'go help'.\n", arg)
	os.Exit(2) // failed at 'go help cmd'
}

var atexitFuncs []func()

func atexit(f func()) {
	atexitFuncs = append(atexitFuncs, f)
}

func exit() {
	for _, f := range atexitFuncs {
		f()
	}
	os.Exit(exitStatus)
}

func fatalf(format string, args ...interface{}) {
	errorf(format, args...)
	exit()
}

func errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
	setExitStatus(1)
}

var logf = log.Printf

func exitIfErrors() {
	if exitStatus != 0 {
		exit()
	}
}
