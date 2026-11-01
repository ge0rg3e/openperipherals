package wallpaper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSources(t *testing.T) {
	file := filepath.Join(t.TempDir(), "scene with spaces.html")
	if err := os.WriteFile(file, []byte("<!doctype html>"), 0600); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(t.TempDir(), "scene with spaces.MP4")
	if err := os.WriteFile(video, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		kind, source string
		valid        bool
	}{
		{"builtin", "", true}, {"local", file, true}, {"local", file + ".js", false},
		{"local", filepath.Join(t.TempDir(), "missing.html"), false},
		{"video", video, true}, {"video", file, false}, {"local", video, false},
		{"video", "", false}, {"video", filepath.Join(t.TempDir(), "missing.mp4"), false},
		{"url", "https://example.com/scene?q=one#scene", true}, {"url", "http://localhost:8000/", true},
		{"url", "file:///C:/secret.html", false}, {"url", "javascript:alert(1)", false},
		{"url", "https://user:password@example.com", false}, {"url", "https://", false},
	} {
		t.Run(test.kind+":"+test.source, func(t *testing.T) {
			c := DefaultConfig()
			c.Kind = test.kind
			c.Source = test.source
			_, err := Validate(c)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, error=%v", test.valid, err)
			}
		})
	}
	dir := filepath.Join(t.TempDir(), "directory.mp4")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	c := DefaultConfig()
	c.Kind, c.Source = "video", dir
	if _, err := Validate(c); err == nil {
		t.Fatal("accepted a directory as an MP4")
	}
	bad := DefaultConfig()
	bad.FPS = 0
	if _, err := Validate(bad); err == nil {
		t.Fatal("accepted unlimited FPS")
	}
}

func TestDesktopCoverage(t *testing.T) {
	primary := [4]int32{0, 0, 1920, 1040}
	for _, test := range []struct {
		window [4]int32
		want   bool
	}{
		{[4]int32{0, 0, 1920, 1080}, true}, {[4]int32{0, 0, 1920, 1040}, true},
		{[4]int32{-8, -8, 1928, 1048}, true}, {[4]int32{100, 100, 1800, 1000}, false},
		{[4]int32{1920, 0, 3840, 1080}, false},
	} {
		if got := Covers(test.window, primary); got != test.want {
			t.Errorf("window=%v: got %v, want %v", test.window, got, test.want)
		}
	}
}
