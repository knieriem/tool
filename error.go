package tool

import (
	"fmt"
	"os"

	"github.com/knieriem/text/line"
)

func PrintErrExit(err error) {
	switch err := err.(type) {
	case *line.ErrorList:
		for _, e := range err.List {
			switch e := e.(type) {
			case line.Error:
				printErr(err.Filename, e.Line(), e.Error())
			default:
				printErr(err.Filename, -1, e.Error())
			}
		}
	case line.Error:
		printErr("", err.Line(), err.Error())
	default:
		printErr("", -1, err.Error())
	}
}

var ErrFormatPrefix string

func printErr(fileName string, line int, msg string) {
	var s string

	s += ErrFormatPrefix
	if fileName != "" {
		fileName = FilepathRelIfShorter(fileName)
		s += fileName + ":"
	}

	if line != -1 {
		s += fmt.Sprintf("%d:", line)
	}

	fmt.Fprintln(os.Stderr, s, msg)
}
