// Command brand writes the icons of every program: the editor's PNG and ICO
// that Wails builds from, and a Windows resource object for each Go program,
// which go build links into its executable. It also copies the engine version
// into wails.json, which Wails reads and Go cannot.
package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/brand"
	ver "gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// programs maps each program to its package and the name Windows shows.
var programs = []struct {
	mark brand.Program
	dir  string
	desc string
}{
	{brand.Launcher, "cmd/ascdii", "ASCDII"},
	{brand.Player, "cmd/player", "ASCDII Player"},
}

func run(ctx context.Context) error {
	if err := writePNG(ctx, "cmd/editor/build/appicon.png", brand.Draw(brand.Editor, 1024)); err != nil {
		return err
	}
	icon, err := icon(brand.Editor)
	if err != nil {
		return err
	}
	if err := writeICO(ctx, "cmd/editor/build/windows/icon.ico", icon); err != nil {
		return err
	}
	for _, p := range programs {
		if err := writeSyso(ctx, p.mark, p.dir, p.desc); err != nil {
			return err
		}
	}
	return writeWailsVersion(ctx, "cmd/editor/wails.json")
}

var productVersion = regexp.MustCompile(`("productVersion":\s*")[^"]*(")`)

// writeWailsVersion edits the one field in place, so the file keeps its layout.
func writeWailsVersion(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("brand version: %w", err)
	}
	out := productVersion.ReplaceAll(b, []byte("${1}"+ver.Engine+"${2}"))
	if string(out) == string(b) {
		return nil
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("brand version: %w", err)
	}
	return nil
}

// icon draws every size Windows asks for at that size, so small ones stay
// sharp instead of being scaled down.
func icon(p brand.Program) (*winres.Icon, error) {
	var imgs []image.Image
	for _, s := range []int{256, 64, 48, 32, 24, 16} {
		imgs = append(imgs, brand.Draw(p, s))
	}
	return winres.NewIconFromImages(imgs)
}

func writeSyso(ctx context.Context, p brand.Program, dir, desc string) error {
	ic, err := icon(p)
	if err != nil {
		return fmt.Errorf("brand %s: %w", p, err)
	}
	rs := &winres.ResourceSet{}
	if err := rs.SetIcon(winres.Name("APPICON"), ic); err != nil {
		return fmt.Errorf("brand %s: %w", p, err)
	}
	rs.SetManifest(winres.AppManifest{DPIAwareness: winres.DPIPerMonitorV2, LongPathAware: true, UseCommonControlsV6: true})
	var vi version.Info
	vi.SetFileVersion(ver.Engine)
	vi.SetProductVersion(ver.Engine)
	for k, v := range map[string]string{version.ProductName: "ASCDII Engine", version.FileDescription: desc} {
		if err := vi.Set(version.LangDefault, k, v); err != nil {
			return fmt.Errorf("brand %s: %w", p, err)
		}
	}
	rs.SetVersionInfo(vi)
	return write(ctx, filepath.Join(dir, "rsrc_windows_amd64.syso"), func(f *os.File) error { return rs.WriteObject(f, winres.ArchAMD64) })
}

func writePNG(ctx context.Context, out string, img image.Image) error {
	return write(ctx, out, func(f *os.File) error { return png.Encode(f, img) })
}

func writeICO(ctx context.Context, out string, ic *winres.Icon) error {
	return write(ctx, out, func(f *os.File) error { return ic.SaveICO(f) })
}

func write(ctx context.Context, out string, fill func(*os.File) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("brand: %w", err)
	}
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("brand: %w", err)
	}
	if err := fill(f); err != nil {
		f.Close()
		return fmt.Errorf("brand %s: %w", out, err)
	}
	return f.Close()
}
