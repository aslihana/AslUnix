//go:build windows

package main

import (
	"io"
	"os"
	"unsafe"
)

func colorMode(out io.Writer, requested bool) (bool, func()) {
	if !requested {
		return false, func() {}
	}
	file, ok := out.(*os.File)
	if !ok {
		return false, func() {}
	}
	var mode uint32
	success, _, _ := getMode.Call(file.Fd(), uintptr(unsafe.Pointer(&mode)))
	if success == 0 {
		return false, func() {}
	}
	success, _, _ = setMode.Call(file.Fd(), uintptr(mode|4))
	if success == 0 {
		return false, func() {}
	}
	return true, func() { setMode.Call(file.Fd(), uintptr(mode)) }
}
