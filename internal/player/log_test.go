package player

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestUnit_HeldLogKeepsLatestAndCountsDropped(t *testing.T) {
	// Arrange
	l := &heldLog{}
	for i := range heldMax + 3 {
		l.Log("error", "scripts/door.lua", fmt.Sprint("line ", i))
	}
	var out bytes.Buffer

	// Act
	l.Flush(&out)

	// Assert
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != heldMax+1 || lines[0] != "(3 earlier messages dropped)" || lines[1] != "error scripts/door.lua: line 3" {
		t.Fatalf("%d lines, first %q %q", len(lines), lines[0], lines[1])
	}
}
