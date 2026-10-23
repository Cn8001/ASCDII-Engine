// Command docref writes a component reference into a docs page, between its
// "docref:begin" and "docref:end" markers: every kind with its properties,
// types, defaults and ranges, and the descriptions in notes.go. Without a
// module it documents the engine's components, with one that module's.
//
//	go run ./cmd/docref docs/usage/components.html
//	go run ./cmd/docref docs/usage/dragonfall.html dragonfall
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

const (
	begin = "<!-- docref:begin -->"
	end   = "<!-- docref:end -->"
)

var errMarkers = errors.New("page has no docref markers")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		return errors.New("usage: docref <page.html> [module]")
	}
	module := ""
	if len(os.Args) == 3 {
		module = os.Args[2]
	}
	page, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	i, j := bytes.Index(page, []byte(begin)), bytes.Index(page, []byte(end))
	if i < 0 || j < i {
		return fmt.Errorf("%s: %w", os.Args[1], errMarkers)
	}
	body, err := reference(module)
	if err != nil {
		return err
	}
	out := append(append(append([]byte{}, page[:i+len(begin)]...), body...), page[j:]...)
	return os.WriteFile(os.Args[1], out, 0o644)
}

func reference(module string) ([]byte, error) {
	// Ordered as the GameObject menu and Add Component list them; a module's
	// components share its name as their category.
	groups := []struct{ category, title string }{
		{"3d", "3D objects"}, {"camera", "Camera"}, {"light", "Lights"}, {"effects", "Effects"}, {"ui", "UI"},
		{"audio", "Audio"}, {"physics", "Physics"}, {"logic", "Logic"},
	}
	if module != "" {
		groups = []struct{ category, title string }{{module, ""}}
	}
	reg, _ := project.Registry([]project.Module{dragonfall.Module()})
	docs := notes()
	var b strings.Builder
	b.WriteString("\n")
	for _, g := range groups {
		if g.title != "" {
			fmt.Fprintf(&b, "<h2>%s</h2>\n", g.title)
		}
		for _, k := range reg.Kinds() {
			m := reg.Meta(k)
			if m.Category != g.category {
				continue
			}
			c, err := reg.New(k)
			if err != nil {
				return nil, err
			}
			n, ok := docs[k]
			if !ok {
				return nil, fmt.Errorf("%s: no note in notes.go", k)
			}
			where := "Add Component"
			if m.Object {
				where = "GameObject menu and Add Component"
			}
			fmt.Fprintf(&b, "<h3 id=\"%s\">%s <code>%s</code></h3>\n<p>%s <span class=\"dim\">%s.</span></p>\n", k, html.EscapeString(m.Label), k, html.EscapeString(n.summary), where)
			b.WriteString("<table>\n<tr><th>Property</th><th>Type</th><th>Default</th><th>Description</th></tr>\n")
			for _, p := range c.Props() {
				text, ok := n.fields[p.Name]
				if !ok {
					return nil, fmt.Errorf("%s.%s: no description in notes.go", k, p.Name)
				}
				fmt.Fprintf(&b, "<tr><td><code>%s</code></td><td>%s</td><td><code>%s</code></td><td>%s</td></tr>\n",
					p.Name, typeOf(p), html.EscapeString(value(p)), html.EscapeString(text))
			}
			b.WriteString("</table>\n")
		}
	}
	return []byte(b.String()), nil
}

func typeOf(p prop.Prop) string {
	t := string(p.Type)
	switch {
	case len(p.Options) > 0 && p.Type == prop.Enum:
		return t + ": " + strings.Join(p.Options, ", ")
	case p.Max > p.Min:
		return fmt.Sprintf("%s, %g to %g", t, p.Min, p.Max)
	}
	return t
}

func value(p prop.Prop) string {
	v := prop.Encode(p)
	if s, ok := v.(string); ok {
		if s == "" {
			return "none"
		}
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return strings.ReplaceAll(string(b), ",", ", ")
}
