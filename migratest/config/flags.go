package config

import (
	"flag"
	"time"
)

type Flags struct {
	Debug        bool
	Config       string
	StartTimeout time.Duration
	StopTimeout  time.Duration
}

//nolint:gomnd // init
func NewFlags() *Flags {
	var f Flags

	flag.BoolVar(&f.Debug, "debug", false, "debug mode")
	flag.StringVar(&f.Config, "config", "mtest.yml", "config file to use")
	flag.DurationVar(&f.StartTimeout, "start-timeout", 5*time.Second, "start timeout")
	flag.DurationVar(&f.StopTimeout, "stop-timeout", 5*time.Second, "stop timeout")
	return &f
}
