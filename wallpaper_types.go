package main

import "openperipherals/internal/wallpaper"

type WallpaperStatus struct {
	Config      wallpaper.Config `json:"config"`
	Phase       string           `json:"phase"`
	Reason      string           `json:"reason"`
	ManualPause bool             `json:"manualPause"`
	Snapshot    bool             `json:"snapshot"`
}
