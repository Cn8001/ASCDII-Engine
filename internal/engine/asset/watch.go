package asset

import (
	"context"
	"io/fs"
	"time"
)

type stamp struct {
	mod  time.Time
	size int64
}

// Watch polls fsys and calls fn with the asset paths that changed, appeared
// or disappeared. Polling keeps it standard library only and works on WSL
// mounts where file events are unreliable. It returns when ctx ends.
func Watch(ctx context.Context, fsys fs.FS, every time.Duration, fn func(changed []string), skip ...string) error {
	seen, err := snapshot(ctx, fsys, skip)
	if err != nil {
		return err
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			now, err := snapshot(ctx, fsys, skip)
			if err != nil {
				continue // a file mid write can fail to stat; try again next tick
			}
			var changed []string
			for p, s := range now {
				if old, ok := seen[p]; !ok || old != s {
					changed = append(changed, p)
				}
			}
			for p := range seen {
				if _, ok := now[p]; !ok {
					changed = append(changed, p)
				}
			}
			seen = now
			if len(changed) > 0 {
				fn(changed)
			}
		}
	}
}

func snapshot(ctx context.Context, fsys fs.FS, skip []string) (map[string]stamp, error) {
	entries, err := List(ctx, fsys, skip...)
	if err != nil {
		return nil, err
	}
	out := make(map[string]stamp, len(entries))
	for _, e := range entries {
		info, err := fs.Stat(fsys, e.Path)
		if err != nil {
			return nil, err
		}
		out[e.Path] = stamp{mod: info.ModTime(), size: info.Size()}
	}
	return out, nil
}
