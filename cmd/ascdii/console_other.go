//go:build !windows

package main

import (
	"fmt"
	"os"
)

func hideConsole() {}

func report(msg string) { fmt.Fprintln(os.Stderr, msg) }
