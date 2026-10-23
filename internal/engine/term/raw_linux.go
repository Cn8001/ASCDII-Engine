//go:build linux

package term

import (
	"fmt"
	"syscall"
	"unsafe"
)

type State struct{ termios syscall.Termios }

func ioctl(fd int, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

func MakeRaw(fd int) (*State, error) {
	var t syscall.Termios
	if err := ioctl(fd, syscall.TCGETS, unsafe.Pointer(&t)); err != nil {
		return nil, fmt.Errorf("get termios: %w", err)
	}
	raw := t
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN], raw.Cc[syscall.VTIME] = 1, 0
	if err := ioctl(fd, syscall.TCSETS, unsafe.Pointer(&raw)); err != nil {
		return nil, fmt.Errorf("set termios: %w", err)
	}
	return &State{termios: t}, nil
}

func Restore(fd int, s *State) error {
	if s == nil {
		return nil
	}
	if err := ioctl(fd, syscall.TCSETS, unsafe.Pointer(&s.termios)); err != nil {
		return fmt.Errorf("restore termios: %w", err)
	}
	return nil
}

type winSize struct{ Rows, Cols, X, Y uint16 }

func Size(fd int) (int, int, error) {
	var ws winSize
	if err := ioctl(fd, syscall.TIOCGWINSZ, unsafe.Pointer(&ws)); err != nil {
		return 0, 0, fmt.Errorf("terminal size: %w", err)
	}
	if ws.Cols == 0 || ws.Rows == 0 {
		return 0, 0, fmt.Errorf("terminal size: %w", ErrNotTerminal)
	}
	return int(ws.Cols), int(ws.Rows), nil
}
