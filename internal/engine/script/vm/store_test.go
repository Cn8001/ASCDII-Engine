package vm

import (
	"context"
	"io/fs"
	"strings"
	"testing"
)

type memStore map[string][]byte

func (m memStore) Load(_ context.Context, slot string) ([]byte, error) {
	data, ok := m[slot]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

func (m memStore) Save(_ context.Context, slot string, data []byte) error {
	m[slot] = data
	return nil
}

func TestUnit_SaveAndLoadKeepPlainData(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/s.lua": `
function start(self)
  save("game", { money = 250, done = { "a", "b" }, home = vec3(1, 2, 3), skip = self })
  local back = load("game")
  log(back.money, back.done[2], back.home.z, back.skip == nil, load("empty") == nil)
end`})
	w.c.Store = memStore{}
	w.s.Add("s", w.script(t, "scripts/s.lua"))

	// Act
	w.run(1)

	// Assert
	if len(w.log.lines) == 0 || !strings.HasSuffix(w.log.lines[len(w.log.lines)-1], " 250 b 3 true true") {
		t.Fatalf("logs %v", w.log.lines)
	}
}

func TestUnit_SaveWithoutStoreReturnsNilAndMessage(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/s.lua": `
function start(self)
  local ok, err = save("game", { a = 1 })
  log(ok == nil, err)
end`})
	w.s.Add("s", w.script(t, "scripts/s.lua"))

	// Act
	w.run(1)

	// Assert
	if len(w.log.lines) == 0 || !strings.HasSuffix(w.log.lines[len(w.log.lines)-1], " true no save store") {
		t.Fatalf("logs %v", w.log.lines)
	}
}
