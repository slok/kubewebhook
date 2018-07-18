package main

import (
	"flag"
	"os"
)

// Defaults.
const (
	lAddressDef     = ":8080"
	lMetricsAddress = ":8081"
	debugDef        = false
)

// Flags are the flags of the program.
type Flags struct {
	ListenAddress        string
	MetricsListenAddress string
	Debug                bool
	CertFile             string
	KeyFile              string
}

// NewFlags returns the flags of the commandline.
func NewFlags() *Flags {
	flags := &Flags{}
	fl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fl.StringVar(&flags.ListenAddress, "listen-address", lAddressDef, "webhook server listen address")
	fl.StringVar(&flags.MetricsListenAddress, "metrics-listen-address", lMetricsAddress, "metrics server listen address")
	fl.BoolVar(&flags.Debug, "debug", debugDef, "enable debug mode")
	fl.StringVar(&flags.CertFile, "tls-cert-file", "certs/cert.pem", "TLS certificate file")
	fl.StringVar(&flags.KeyFile, "tls-key-file", "certs/key.pem", "TLS key file")

	fl.Parse(os.Args[1:])

	return flags
}
