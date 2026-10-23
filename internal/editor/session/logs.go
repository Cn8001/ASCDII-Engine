package session

import (
	"context"
	"sync"
	"time"
)

const LogEvent = "log"

type LogEntry struct {
	Time   string `json:"time"`
	Level  string `json:"level"`
	Source string `json:"source"`
	Text   string `json:"text"`
	Count  int    `json:"count"`
}

// logbook keeps recent console lines and collapses repeats like Unity's
// collapse toggle, so a script logging every frame cannot flood the UI.
type logbook struct {
	mu      sync.Mutex
	entries []LogEntry
}

const logLimit = 300

func (b *logbook) add(e LogEntry) (LogEntry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n := len(b.entries); n > 0 {
		last := &b.entries[n-1]
		if last.Level == e.Level && last.Source == e.Source && last.Text == e.Text {
			last.Count++
			last.Time = e.Time
			return *last, false
		}
	}
	e.Count = 1
	b.entries = append(b.entries, e)
	if len(b.entries) > logLimit {
		b.entries = b.entries[len(b.entries)-logLimit:]
	}
	return e, true
}

func (b *logbook) list() []LogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]LogEntry{}, b.entries...)
}

func (b *logbook) clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = nil
}

// Log records a line and pushes it to the UI. It is safe without the session lock.
func (s *Session) Log(ctx context.Context, level, source, text string) {
	e, _ := s.logs.add(LogEntry{Time: s.d.Now().Format(time.TimeOnly), Level: level, Source: source, Text: text})
	if s.d.Emit != nil {
		s.d.Emit.Emit(ctx, LogEvent, e)
	}
}

func (s *Session) Logs() []LogEntry { return s.logs.list() }

func (s *Session) ClearLogs() { s.logs.clear() }
