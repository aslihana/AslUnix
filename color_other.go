//go:build !windows

package main

import "io"

func colorMode(out io.Writer, requested bool) (bool, func()) { return requested, func() {} }
