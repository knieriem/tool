package tool

import (
	"fmt"
	"io"
)

var Version = "<not set>"

var CmdVersion = &Command{
	UsageLine: "version",
	Short:     "displays the program version",
	Run: func(_ *Command, _ io.Writer, _ []string) error {
		fmt.Println(Name, "version", Version)
		return nil
	},
}
