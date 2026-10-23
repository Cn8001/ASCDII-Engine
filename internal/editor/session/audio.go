package session

import (
	"fmt"
	"os"
	"path/filepath"
)

// AudioEvent tells the editor's web view what to play; it fetches the file
// from /asset/<path>, so sounds decode in the browser, not in Go.
type AudioEvent struct {
	Op     string  `json:"op"` // play, stop, volume, stopall
	Voice  uint64  `json:"voice"`
	Path   string  `json:"path,omitempty"`
	Volume float64 `json:"volume"`
	Loop   bool    `json:"loop,omitempty"`
}

// editorAudio is the AudioOut of play mode. It runs under the session lock,
// like the game it serves.
type editorAudio struct {
	s    *Session
	next uint64
	live map[uint64]bool
}

func (a *editorAudio) emit(ev AudioEvent) {
	if a.s.d.Emit != nil {
		a.s.d.Emit.Emit(a.s.appCtx, "audio", ev)
	}
}

func (a *editorAudio) Play(ref string, volume float64, loop bool) (uint64, error) {
	p := a.s.lib.Resolve(ref)
	if _, err := os.Stat(filepath.Join(a.s.dir, filepath.FromSlash(p))); err != nil {
		return 0, fmt.Errorf("sound %s: %w", p, err)
	}
	a.next++
	a.live[a.next] = true
	a.emit(AudioEvent{Op: "play", Voice: a.next, Path: p, Volume: volume, Loop: loop})
	return a.next, nil
}

func (a *editorAudio) Stop(v uint64) {
	delete(a.live, v)
	a.emit(AudioEvent{Op: "stop", Voice: v})
}

func (a *editorAudio) SetVolume(v uint64, vol float64) {
	a.emit(AudioEvent{Op: "volume", Voice: v, Volume: vol})
}

func (a *editorAudio) Playing(v uint64) bool { return a.live[v] }

// AudioEnded is how the web view reports a sound that played to its end.
func (s *Session) AudioEnded(v uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.sctx.Audio.(*editorAudio); ok {
		delete(a.live, v)
	}
}

// AssetFile is the file behind a project path, for the /asset/ route; paths
// outside the project are refused.
// Path is where a project file or folder lies on disk; "" is the project
// folder itself.
func (s *Session) Path(rel string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	rel, err := clean(rel)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.dir, filepath.FromSlash(rel)), nil
}

func (s *Session) AssetFile(rel string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	rel, err := clean(s.lib.Resolve(rel))
	if err != nil || rel == "" {
		return "", fmt.Errorf("asset %q: %w", rel, ErrBadPath)
	}
	return filepath.Join(s.dir, filepath.FromSlash(rel)), nil
}
