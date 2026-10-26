package main

import "openperipherals/internal/sysstats"

// SystemService reports CPU, RAM and GPU load. The UI polls it only while the
// window is visible, so it costs nothing in the tray.
type SystemService struct {
	sampler sysstats.Sampler
}

// Stats samples current usage. CPU usage covers the time since the last call.
func (s *SystemService) Stats() sysstats.Stats {
	return s.sampler.Sample()
}
