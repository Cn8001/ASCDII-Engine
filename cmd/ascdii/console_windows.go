//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// ownConsole reports whether Windows opened the console for this process
// alone, as for a double click; a terminal's console has the shell attached.
func ownConsole() bool {
	var pids [2]uint32
	n, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleProcessList").Call(uintptr(unsafe.Pointer(&pids[0])), 2)
	return n == 1
}

// hideConsole closes a console of our own, so starting the editor leaves no
// empty window behind.
func hideConsole() {
	if !ownConsole() {
		return
	}
	kernel32, user32 := syscall.NewLazyDLL("kernel32.dll"), syscall.NewLazyDLL("user32.dll")
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
