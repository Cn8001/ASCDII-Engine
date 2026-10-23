// Package pack handles packages: sets of assets with their .meta files and
// a pack.json, shared as one .ascdiipack file, like Unity packages. They are
// installed into a project's packages folder and are read only there; the
// only operations are install and remove.
package pack

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	Ext          = ".ascdiipack"
	ManifestFile = "pack.json"
	Dir          = "packages" // inside a project
)

var (
	ErrBadName  = errors.New("package names are lower case letters, digits, dots and dashes")
	ErrNotPack  = errors.New("not a package")
	ErrConflict = errors.New("package assets clash with project assets")
)

type Manifest struct {
	Format      int      `json:"format"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Engine      string   `json:"engine,omitempty"` // engine version it was made with
	Description string   `json:"description,omitempty"`
	Author      string   `json:"author,omitempty"`
	Modules     []string `json:"modules,omitempty"` // compiled Go components the assets use
}

func ValidName(n string) bool { return regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`).MatchString(n) }

// Source is a package ready to install: from a file, a folder or built into
// the editor. Origin says which, for the package manager.
type Source struct {
	Manifest Manifest
	FS       fs.FS
	Origin   string
}

// skip leaves out what belongs to a project or a build, not to a package,
// so a whole project folder can serve as a package source.
func skip(p string, d fs.DirEntry) bool {
	base := path.Base(p)
	if p == "." {
		return false
	}
	if strings.HasPrefix(base, ".") || p == "build" || p == Dir {
		return true
	}
	return !d.IsDir() && (base == ManifestFile || strings.HasSuffix(base, ".ascdiiproj"))
}

func readManifest(fsys fs.FS) (Manifest, error) {
	b, err := fs.ReadFile(fsys, ManifestFile)
	if err != nil {
		return Manifest{}, fmt.Errorf("%w: %w", ErrNotPack, err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("%w: %w", ErrNotPack, err)
	}
	if !ValidName(m.Name) {
		return Manifest{}, fmt.Errorf("%q: %w", m.Name, ErrBadName)
	}
	return m, nil
}

// Open reads a package from a .ascdiipack file or a folder with a pack.json.
// Close the returned closer when done; it is a no-op for folders.
func Open(p string) (Source, io.Closer, error) {
	info, err := os.Stat(p)
	if err != nil {
		return Source{}, nil, fmt.Errorf("open package: %w", err)
	}
	var fsys fs.FS
	var closer io.Closer = io.NopCloser(nil)
	if info.IsDir() {
		fsys = os.DirFS(p)
	} else {
		z, err := zip.OpenReader(p)
		if err != nil {
			return Source{}, nil, fmt.Errorf("open package %s: %w", filepath.Base(p), err)
		}
		fsys, closer = z, z
	}
	m, err := readManifest(fsys)
	if err != nil {
		_ = closer.Close()
		return Source{}, nil, err
	}
	return Source{Manifest: m, FS: fsys, Origin: p}, closer, nil
}

// Files lists what a package installs, slash separated and sorted.
func Files(fsys fs.FS) ([]string, error) {
	var out []string
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip(p, d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list package: %w", err)
	}
	sort.Strings(out)
	return out, nil
}

// guidOf reads the id from a .meta file without the asset package, which
// would import this one.
func guidOf(b []byte) string {
	var m struct {
		GUID string `json:"guid"`
	}
	_ = json.Unmarshal(b, &m)
	return m.GUID
}

// Install copies src into dir/packages/<name>, replacing an older version.
// owner tells where the project already has an asset with a GUID; a GUID
// owned outside this package is a clash, since it would name two assets.
func Install(ctx context.Context, dir string, src Source, owner func(guid string) (string, bool)) error {
	name := src.Manifest.Name
	if !ValidName(name) {
		return fmt.Errorf("%q: %w", name, ErrBadName)
	}
	files, err := Files(src.FS)
	if err != nil {
		return err
	}
	home := Dir + "/" + name + "/"
	var clashes []string
	for _, f := range files {
		if !strings.HasSuffix(f, ".meta") {
			continue
		}
		b, err := fs.ReadFile(src.FS, f)
		if err != nil {
			return fmt.Errorf("install %s: %w", name, err)
		}
		if p, ok := owner(guidOf(b)); ok && !strings.HasPrefix(p, home) {
			clashes = append(clashes, p)
		}
	}
	if len(clashes) > 0 {
		return fmt.Errorf("install %s: %w: %s", name, ErrConflict, strings.Join(clashes, ", "))
	}
	tmp := filepath.Join(dir, Dir, "."+name+".installing")
	if err := os.RemoveAll(tmp); err != nil {
		return fmt.Errorf("install %s: %w", name, err)
	}
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			_ = os.RemoveAll(tmp)
			return err
		}
		if err := copyFile(src.FS, f, filepath.Join(tmp, filepath.FromSlash(f))); err != nil {
			_ = os.RemoveAll(tmp)
			return fmt.Errorf("install %s: %w", name, err)
		}
	}
	m := src.Manifest
	if m.Format == 0 {
		m.Format = 1
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(tmp, ManifestFile), append(b, '\n'), 0o644); err != nil {
		_ = os.RemoveAll(tmp)
		return fmt.Errorf("install %s: %w", name, err)
	}
	final := filepath.Join(dir, Dir, name)
	if err := os.RemoveAll(final); err != nil {
		return fmt.Errorf("install %s: %w", name, err)
	}
	if err := os.Rename(tmp, final); err != nil {
		return fmt.Errorf("install %s: %w", name, err)
	}
	return nil
}

func copyFile(fsys fs.FS, from, to string) error {
	in, err := fsys.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func Remove(ctx context.Context, dir, name string) error {
	if !ValidName(name) {
		return fmt.Errorf("%q: %w", name, ErrBadName)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(dir, Dir, name)); err != nil {
		return fmt.Errorf("remove %s: %w", name, err)
	}
	return nil
}

// Installed lists the packages of a project folder.
func Installed(fsys fs.FS) ([]Manifest, error) {
	entries, err := fs.ReadDir(fsys, Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}
	var out []Manifest
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		sub, err := fs.Sub(fsys, Dir+"/"+e.Name())
		if err != nil {
			continue
		}
		if m, err := readManifest(sub); err == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

// Export writes the files of fsys, metas included, with m as a .ascdiipack.
func Export(ctx context.Context, w io.Writer, fsys fs.FS, m Manifest) error {
	if !ValidName(m.Name) {
		return fmt.Errorf("%q: %w", m.Name, ErrBadName)
	}
	files, err := Files(fsys)
	if err != nil {
		return err
	}
	z := zip.NewWriter(w)
	if m.Format == 0 {
		m.Format = 1
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	mw, err := z.Create(ManifestFile)
	if err != nil {
		return fmt.Errorf("export %s: %w", m.Name, err)
	}
	if _, err := mw.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("export %s: %w", m.Name, err)
	}
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		fw, err := z.Create(f)
		if err != nil {
			return fmt.Errorf("export %s: %w", m.Name, err)
		}
		in, err := fsys.Open(f)
		if err != nil {
			return fmt.Errorf("export %s: %w", m.Name, err)
		}
		_, err = io.Copy(fw, in)
		_ = in.Close()
		if err != nil {
			return fmt.Errorf("export %s: %w", m.Name, err)
		}
	}
	if err := z.Close(); err != nil {
		return fmt.Errorf("export %s: %w", m.Name, err)
	}
	return nil
}
