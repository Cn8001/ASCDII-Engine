//go:build !linux && !windows

package terminal

import "fmt"

type State struct{}

func MakeRaw(int) (*State, error) { return nil, fmt.Errorf("raw mode: %w", ErrNotTerminal) }
func Restore(int, *State) error   { return nil }
func Size(int) (int, int, error)  { return 0, 0, fmt.Errorf("terminal size: %w", ErrNotTerminal) }
