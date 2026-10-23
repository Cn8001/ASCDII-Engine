//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// hideConsole closes the console Windows opened for a double click; one
// shared with a terminal (more than this process attached) stays.
func hideConsole() {
	kernel32, user32 := syscall.NewLazyDLL("kernel32.dll"), syscall.NewLazyDLL("user32.dll")
	var pids [2]uint32
	n, _, _ := kernel32.NewProc("GetConsoleProcessList").Call(uintptr(unsafe.Pointer(&pids[0])), 2)
	if n != 1 {
		return
	}
	const swHide = 0
	if w, _, _ := kernel32.NewProc("GetConsoleWindow").Call(); w != 0 {
		_, _, _ = user32.NewProc("ShowWindow").Call(w, swHide)
	}
	_, _, _ = kernel32.NewProc("FreeConsole").Call()
}

// report shows msg in a message box, since a double click has no console
// left to print to.
func report(msg string) {
	text, err := syscall.UTF16PtrFromString(msg)
	if err != nil {
		return
	}
	title, _ := syscall.UTF16PtrFromString("ASCDII")
	const iconError = 0x10
	_, _, _ = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), iconError)
}
