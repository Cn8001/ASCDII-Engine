package player

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

const modLogFile = "log.txt"

// modsDir is the mods folder as the engine reads it, nil when there is none.
func modsDir(dir string) fs.FS {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil
	}
	return os.DirFS(dir)
}

// fileLog writes messages to mods/log.txt; the terminal shows the game, so
// modders read their log there. The file is created on the first message,
// so a run without messages leaves the previous log in place.
type fileLog struct {
	path string
	once sync.Once
	mu   sync.Mutex
	w    io.WriteCloser
}

func newFileLog(dir string) *fileLog { return &fileLog{path: filepath.Join(dir, modLogFile)} }

func (l *fileLog) Log(level, source, msg string) {
	l.once.Do(func() {
		if f, err := os.Create(l.path); err == nil {
			l.w = f
		}
	})
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.w != nil {
		fmt.Fprintf(l.w, "%s %s: %s\n", level, source, msg)
	}
}

func (l *fileLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.w == nil {
		return nil
	}
	return l.w.Close()
}

func stderrLog(level, source, msg string) { fmt.Fprintf(os.Stderr, "%s %s: %s\n", level, source, msg) }
