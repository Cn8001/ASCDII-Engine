package brand

import "testing"

func TestUnit_IconHasTransparentCornersAndAmberDots(t *testing.T) {
	// Arrange
	size := 128

	// Act
	img := Icon(size)

	// Assert
	if a := img.RGBAAt(0, 0).A; a != 0 {
		t.Fatalf("corner alpha %d", a)
	}
	dot := img.RGBAAt(int(.62*float64(size)), int(.2*float64(size)))
	if dot.R < 200 || dot.G < 150 || dot.B > 140 {
		t.Fatalf("lit dot %+v", dot)
	}
	off := img.RGBAAt(int(.38*float64(size)), int(.2*float64(size)))
	if off.R > 80 {
		t.Fatalf("unlit dot %+v", off)
	}
}

func TestUnit_ProgramsHaveDistinctBadges(t *testing.T) {
	// Arrange
	size := 128
	seen := map[[3]uint8]Program{}

	// Act
	for _, p := range Programs() {
		c := Draw(p, size).RGBAAt(int(.77*float64(size)), int(.62*float64(size)))
		seen[[3]uint8{c.R, c.G, c.B}] = p
	}

	// Assert
	if len(seen) != len(Programs()) {
		t.Fatalf("badges share colors: %v", seen)
	}
}
