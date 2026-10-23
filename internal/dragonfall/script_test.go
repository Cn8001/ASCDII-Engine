package dragonfall

import (
	"strings"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func TestUnit_DragonScriptHintsNearby(t *testing.T) {
	// Arrange
	e := open(t)
	var logs []string
	s := e.Scene()
	s.Find("hero").Local.Pos = geom.V3(136, 76, 0)
	c := &scene.Ctx{Scene: s, Log: func(l, src, m string) { logs = append(logs, l+" "+src+" "+m) }}

	// Act
	for i := 0; i < 3; i++ {
		if err := s.Update(c, 1.0/30, nil); err != nil {
			t.Fatal(err)
		}
	}

	// Assert
	said := false
	for _, ch := range s.Find("dragon").Children {
		if ch.Name == "say" {
			said = true
		}
	}
	if !said || len(logs) != 0 {
		t.Fatalf("said %v logs %s", said, strings.Join(logs, "; "))
	}
}
