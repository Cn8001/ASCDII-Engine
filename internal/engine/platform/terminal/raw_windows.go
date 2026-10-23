//go:build windows

package terminal

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	enableProcessedInput  = 0x1
	enableLineInput       = 0x2
	enableEchoInput       = 0x4
	enableVTInput         = 0x200
	enableProcessedOutput = 0x1
	enableVTProcessing    = 0x4
	utf8CodePage          = 65001
)

type State struct {
	inMode, outMode uint32
	codePage        uintptr
}

func kernel(name string) *syscall.LazyProc { return syscall.NewLazyDLL("kernel32.dll").NewProc(name) }

func setMode(h syscall.Handle, mode uint32) error {
	if r, _, err := kernel("SetConsoleMode").Call(uintptr(h), uintptr(mode)); r == 0 {
		return err
	}
	return nil
}

// MakeRaw switches the console to virtual terminal input and output so the
// same escape sequences work as on Linux.
func MakeRaw(fd int) (*State, error) {
	in, out := syscall.Handle(fd), syscall.Stdout
	s := &State{}
	if err := syscall.GetConsoleMode(in, &s.inMode); err != nil {
		return nil, fmt.Errorf("get console input mode: %w", err)
	}
	if err := syscall.GetConsoleMode(out, &s.outMode); err != nil {
		return nil, fmt.Errorf("get console output mode: %w", err)
	}
	s.codePage, _, _ = kernel("GetConsoleOutputCP").Call()
	if err := setMode(in, s.inMode&^(enableProcessedInput|enableLineInput|enableEchoInput)|enableVTInput); err != nil {
		return nil, fmt.Errorf("set console input mode: %w", err)
	}
	if err := setMode(out, s.outMode|enableProcessedOutput|enableVTProcessing); err != nil {
		return nil, fmt.Errorf("set console output mode: %w", err)
	}
	kernel("SetConsoleOutputCP").Call(utf8CodePage)
	return s, nil
}

func Restore(fd int, s *State) error {
	if s == nil {
		return nil
	}
	kernel("SetConsoleOutputCP").Call(s.codePage)
	if err := setMode(syscall.Stdout, s.outMode); err != nil {
		return fmt.Errorf("restore console output: %w", err)
	}
	if err := setMode(syscall.Handle(fd), s.inMode); err != nil {
		return fmt.Errorf("restore console input: %w", err)
	}
	return nil
}

type coord struct{ X, Y int16 }

type smallRect struct{ Left, Top, Right, Bottom int16 }

type bufferInfo struct {
	Size       coord
	Cursor     coord
	Attributes uint16
	Window     smallRect
	MaxWindow  coord
}

func Size(int) (int, int, error) {
	var info bufferInfo
	if r, _, err := kernel("GetConsoleScreenBufferInfo").Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&info))); r == 0 {
		return 0, 0, fmt.Errorf("terminal size: %w", err)
	}
	return int(info.Window.Right-info.Window.Left) + 1, int(info.Window.Bottom-info.Window.Top) + 1, nil
}
