package wallpaper

import (
	"image"
	"image/color"
	"testing"
)

func TestSnapshotPixelsAndMemoryBound(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})
	s := makeSnapshot(img)
	if s.width != 2 || s.height != 2 || s.pixels[2] != 255 || s.pixels[8] != 255 {
		t.Fatal("snapshot does not store top-down BGRA pixels")
	}
	for _, size := range [][2]int{{3840, 1}, {100, 4000}} {
		still := makeSnapshot(image.NewRGBA(image.Rect(0, 0, size[0], size[1])))
		if still.width > 1920 || still.height > 1080 || len(still.pixels) > 1920*1080*4 {
			t.Fatal("snapshot exceeds 1080p memory bound")
		}
	}
}
