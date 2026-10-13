//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// Only brand an independently launched legacy console. A shell-owned console
// must keep its title/icon, and Windows Terminal owns its own visible window.
func brandStandaloneConsole() {
	processList := kernel.NewProc("GetConsoleProcessList")
	var processes [2]uint32
	count, _, _ := processList.Call(uintptr(unsafe.Pointer(&processes[0])), uintptr(len(processes)))
	if count != 1 {
		return
	}
	window, _, _ := kernel.NewProc("GetConsoleWindow").Call()
	if window == 0 {
		return
	}
	user := syscall.NewLazyDLL("user32.dll")
	visible, _, _ := user.NewProc("IsWindowVisible").Call(window)
	if visible == 0 {
		return
	}
	title, _ := syscall.UTF16PtrFromString("AslUnix")
	kernel.NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(title)))
	module, _, _ := kernel.NewProc("GetModuleHandleW").Call(0)
	if module == 0 {
		return
	}
	load := user.NewProc("LoadImageW")
	send := user.NewProc("SendMessageTimeoutW")
	for _, entry := range []struct{ kind, size uintptr }{{0, 16}, {1, 32}} {
		// Resource group 1; IMAGE_ICON=1, LR_SHARED=0x8000. Shared handles need no cleanup.
		icon, _, _ := load.Call(module, 1, 1, entry.size, entry.size, 0x8000)
		if icon == 0 {
			continue
		}
		var previous uintptr
		// WM_SETICON=0x80; SMTO_ABORTIFHUNG=2, bounded to 100ms.
		send.Call(window, 0x80, entry.kind, icon, 2, 100, uintptr(unsafe.Pointer(&previous)))
	}
}
