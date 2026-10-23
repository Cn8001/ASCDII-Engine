package service

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/session"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/pack"
)

// PackageService is the package manager: list, install, remove and export.
type PackageService struct{ d Deps }

func (p *PackageService) List() (session.Packages, error) {
	return wrap1(p.d.Session.PackageList(p.d.Ctx.Get()))
}

// Install takes a source from List: "builtin:<name>" or a file path.
func (p *PackageService) Install(source string) (string, error) {
	return wrap1(p.d.Session.InstallPackage(p.d.Ctx.Get(), source))
}

// InstallFile asks for a .ascdiipack and installs it.
func (p *PackageService) InstallFile() (string, error) {
	ctx := p.d.Ctx.Get()
	file, err := p.d.Dialogs.OpenFile(ctx, "Install package", []Filter{{Name: "ASCDII package", Pattern: "*" + pack.Ext}})
	if err != nil {
		return "", wrap(err)
	}
	if file == "" {
		return "", wrap(errCancelled)
	}
	return wrap1(p.d.Session.InstallPackage(ctx, file))
}

func (p *PackageService) Remove(name string) error {
	return wrap(p.d.Session.RemovePackage(p.d.Ctx.Get(), name))
}

// Export saves a project folder as a .ascdiipack where the user picks.
func (p *PackageService) Export(folder string, m pack.Manifest) (string, error) {
	ctx := p.d.Ctx.Get()
	if !pack.ValidName(m.Name) {
		return "", wrap(pack.ErrBadName)
	}
	out, err := p.d.Dialogs.SaveFile(ctx, "Export package", m.Name+"-"+m.Version+pack.Ext, []Filter{{Name: "ASCDII package", Pattern: "*" + pack.Ext}})
	if err != nil {
		return "", wrap(err)
	}
	if out == "" {
		return "", wrap(errCancelled)
	}
	if err := p.d.Session.ExportPackage(ctx, folder, m, out); err != nil {
		return "", wrap(err)
	}
	return out, nil
}
