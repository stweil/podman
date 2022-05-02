//go:build !linux
// +build !linux

package main

import (
	"fmt"
	"os"
)

func syslogHook() {
	if !useSyslog {
		return
	}

	fmt.Fprintf(os.Stderr, "Logging to Syslog is not supported on Windows")
	os.Exit(1)
}
