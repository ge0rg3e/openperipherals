package wallpaper

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config is saved by the app, and passed to the isolated wallpaper host over stdin.
type Config struct {
	Enabled      bool   `json:"enabled"`
	Kind         string `json:"kind"`
	Source       string `json:"source"`
	FPS          int    `json:"fps"`
	Span         bool   `json:"span"`
	PauseCovered bool   `json:"pauseCovered"`
	PauseBattery bool   `json:"pauseBattery"`
}

type State struct {
	Phase    string `json:"phase"`
	Reason   string `json:"reason"`
	Snapshot bool   `json:"snapshot"`
}

type Command struct {
	Action string `json:"action"`
	Config Config `json:"config"`
}

func DefaultConfig() Config {
	return Config{Kind: "builtin", FPS: 30, PauseCovered: true, PauseBattery: true}
}

func Validate(c Config) (Config, error) {
	if c.FPS != 15 && c.FPS != 24 && c.FPS != 30 && c.FPS != 60 {
		return c, fmt.Errorf("choose an animation limit of 15, 24, 30 or 60 FPS")
	}
	c.Source = strings.TrimSpace(c.Source)
	switch c.Kind {
	case "builtin":
		c.Source = ""
	case "url":
		u, err := url.Parse(c.Source)
		if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
			return c, fmt.Errorf("enter an HTTP or HTTPS URL without embedded credentials")
		}
	case "local", "video":
		label, extensions := "HTML", ".html or .htm"
		if c.Kind == "video" {
			label, extensions = "MP4", ".mp4"
		}
		path, err := filepath.Abs(c.Source)
		if err != nil || c.Source == "" {
			return c, fmt.Errorf("choose a local %s file", label)
		}
		ext := strings.ToLower(filepath.Ext(path))
		if (c.Kind == "video" && ext != ".mp4") || (c.Kind == "local" && ext != ".html" && ext != ".htm") {
			return c, fmt.Errorf("the wallpaper must be an %s file", extensions)
		}
		info, err := os.Stat(path)
		if err != nil {
			return c, fmt.Errorf("open wallpaper: %w", err)
		}
		if !info.Mode().IsRegular() {
			return c, fmt.Errorf("the wallpaper must be a regular %s file", label)
		}
		c.Source = path
	default:
		return c, fmt.Errorf("unknown wallpaper source")
	}
	return c, nil
}

// Covers accepts a maximised window or a full-screen window, with a small
// allowance for invisible frame borders. It doesn't pause for an adjacent screen.
func Covers(window, target [4]int32) bool {
	const slop = 2
	return window[0] <= target[0]+slop && window[1] <= target[1]+slop &&
		window[2] >= target[2]-slop && window[3] >= target[3]-slop
}
