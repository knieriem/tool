package tool

import (
	"bytes"
	"flag"
)

func FlagDefaults(addFlags func(*flag.FlagSet)) string {
	var f flag.FlagSet
	var b bytes.Buffer

	addFlags(&f)
	f.SetOutput(&b)
	f.PrintDefaults()
	return b.String()
}
