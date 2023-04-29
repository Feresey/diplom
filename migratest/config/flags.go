package config

import "flag"

type Flags struct {
	Debug bool
}

func NewFlags() *Flags {
	var f Flags

	flag.BoolVar(&f.Debug, "debug", false, "debug mode")
	return &f
}
