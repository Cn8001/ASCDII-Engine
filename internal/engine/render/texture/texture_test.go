package texture

import (
	"image"
	stdcolor "image/color"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
)

// checker is black and white texels with a transparent top left quarter.
func checker(n int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			c := stdcolor.NRGBA{A: 255}
			if (x+y)%2 == 0 {
				c = stdcolor.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}
			if x < n/2 && y < n/2 {
				c.A = 0
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func TestUnit_MipsAverageAndShrink(t *testing.T) {
	// Arrange
	img := checker(64)

	// Act
	tx := FromImage(img, Options{MaxSize: 32, Filter: Nearest})
	w, h := tx.Size()
	far, _ := tx.Sample(3, .75, .25)

	// Assert
	if w != 32 || h != 32 || tx.Levels() != 6 {
		t.Fatalf("size %dx%d levels %d", w, h, tx.Levels())
	}
	if far.R < 100 || far.R > 155 {
		t.Fatalf("far mip should be gray, got %+v", far)
	}
}

func TestUnit_SampleWrapsAndCutsHoles(t *testing.T) {
	// Arrange
	tx := FromImage(checker(4), Options{Filter: Nearest})

	// Act
	_, hole := tx.Sample(0, .1, .9) // top left in image space, v grows upward
	a, okA := tx.Sample(0, .6, .1)
	b, okB := tx.Sample(0, 1.6, -.9)

	// Assert
	if hole || !okA || !okB || a != b {
		t.Fatalf("hole %v a %+v b %+v", hole, a, b)
	}
}

func TestUnit_LinearBlendsNeighbours(t *testing.T) {
	// Arrange
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, stdcolor.NRGBA{A: 255})
	img.Set(1, 0, stdcolor.NRGBA{R: 200, G: 200, B: 200, A: 255})
	tx := FromImage(img, Options{Filter: Linear})

	// Act
	mid, _ := tx.Sample(0, .5, .5)

	// Assert
	if mid.R < 80 || mid.R > 120 || mid == (color.Color{}) {
		t.Fatalf("mid %+v", mid)
	}
}
