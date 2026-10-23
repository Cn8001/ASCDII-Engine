package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/pack"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
)

const packUsage = `usage:
  ascdii pack list <project>
  ascdii pack install <project> <file.ascdiipack | dragonfall>
  ascdii pack remove <project> <name>
  ascdii pack export <folder> -name N [-version 1.0.0] [-description D] [-out file]`

func builtins() []pack.Source { return []pack.Source{dragonfall.Package()} }

func cmdPack(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return errors.New(packUsage)
	}
	switch args[0] {
	case "list":
		list, err := pack.Installed(os.DirFS(args[1]))
		if err != nil {
			return err
		}
		for _, m := range list {
			fmt.Printf("%s %s\t%s\n", m.Name, m.Version, m.Description)
		}
		for _, b := range builtins() {
			if !slices.ContainsFunc(list, func(m pack.Manifest) bool { return m.Name == b.Manifest.Name }) {
				fmt.Printf("%s %s\t(built in, not installed)\n", b.Manifest.Name, b.Manifest.Version)
			}
		}
		return nil
	case "install":
		if len(args) < 3 {
			return errors.New(packUsage)
		}
		return packInstall(ctx, args[1], args[2])
	case "remove":
		if len(args) < 3 {
			return errors.New(packUsage)
		}
		return packRemove(ctx, args[1], args[2])
	case "export":
		return packExport(ctx, args[1:])
	}
	return errors.New(packUsage)
}

func packInstall(ctx context.Context, dir, source string) error {
	src, closer, err := packSource(source)
	if err != nil {
		return err
	}
	defer closer()
	p, err := project.Open(ctx, os.DirFS(dir), modules())
	if err != nil {
		return err
	}
	owner := func(guid string) (string, bool) {
		if !p.Library.Known(guid) {
			return "", false
		}
		return p.Library.Resolve(guid), true
	}
	if err := pack.Install(ctx, dir, src, owner); err != nil {
		return err
	}
	added := false
	for _, need := range src.Manifest.Modules {
		if slices.Contains(p.Manifest.Modules, need) {
			continue
		}
		if !slices.ContainsFunc(modules(), func(m project.Module) bool { return m.Name == need }) {
			return fmt.Errorf("package %s: %w: %s", src.Manifest.Name, project.ErrMissingModule, need)
		}
		p.Manifest.Modules, added = append(p.Manifest.Modules, need), true
	}
	if added {
		if err := project.SaveManifest(ctx, filepath.Join(dir, p.File), p.Manifest); err != nil {
			return err
		}
	}
	fmt.Println("installed", src.Manifest.Name, src.Manifest.Version, "into", filepath.Join(dir, pack.Dir, src.Manifest.Name))
	return nil
}

func packSource(source string) (pack.Source, func(), error) {
	if !strings.HasSuffix(source, pack.Ext) {
		for _, b := range builtins() {
			if b.Manifest.Name == source {
				return b, func() {}, nil
			}
		}
	}
	src, closer, err := pack.Open(source)
	if err != nil {
		return pack.Source{}, nil, err
	}
	return src, func() { _ = closer.Close() }, nil
}

func packExport(ctx context.Context, args []string) error {
	folder, rest := split(args)
	fl := flag.NewFlagSet("pack export", flag.ContinueOnError)
	name := fl.String("name", "", "package name: lower case letters, digits, dots, dashes")
	ver := fl.String("version", "1.0.0", "package version")
	desc := fl.String("description", "", "one line description")
	out := fl.String("out", "", "output file, <name>-<version>.ascdiipack by default")
	if err := fl.Parse(rest); err != nil {
		return err
	}
	if folder == "" || *name == "" {
		return errors.New(packUsage)
	}
	m := pack.Manifest{Format: 1, Name: *name, Version: *ver, Engine: version.Engine, Description: *desc}
	var b bytes.Buffer
	if err := pack.Export(ctx, &b, os.DirFS(folder), m); err != nil {
		return err
	}
	file := *out
	if file == "" {
		file = *name + "-" + *ver + pack.Ext
	}
	if err := os.WriteFile(file, b.Bytes(), 0o644); err != nil {
		return fmt.Errorf("export: %w", err)
	}
	fmt.Println("wrote", file)
	return nil
}

// packRemove deletes a package and the modules only it needed.
func packRemove(ctx context.Context, dir, name string) error {
	installed, err := pack.Installed(os.DirFS(dir))
	if err != nil {
		return err
	}
	i := slices.IndexFunc(installed, func(m pack.Manifest) bool { return m.Name == name })
	if i < 0 {
		return fmt.Errorf("%s is not installed", name)
	}
	if err := pack.Remove(ctx, dir, name); err != nil {
		return err
	}
	drop, err := project.Unneeded(ctx, os.DirFS(dir), modules(), installed[i])
	if err != nil || len(drop) == 0 {
		return err
	}
	file, err := project.Find(os.DirFS(dir))
	if err != nil {
		return err
	}
	m, err := project.ReadManifest(os.DirFS(dir), file)
	if err != nil {
		return err
	}
	m.Modules = slices.DeleteFunc(m.Modules, func(x string) bool { return slices.Contains(drop, x) })
	return project.SaveManifest(ctx, filepath.Join(dir, file), m)
}
