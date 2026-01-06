//go:build screen

package screens

import "goratt/video/screen"

// ShutdownScreen displays a blank/shutdown state.
type ShutdownScreen struct {
	mgr *screen.Manager
}

// NewShutdownScreen creates a new shutdown screen.
func NewShutdownScreen() *ShutdownScreen {
	return &ShutdownScreen{}
}

func (s *ShutdownScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}

func (s *ShutdownScreen) Update() {
	s.mgr.FillBackground(0, 0, 0) // Black
	s.mgr.Flush()
}

func (s *ShutdownScreen) HandleEvent(event screen.Event) bool {
	return false
}

func (s *ShutdownScreen) Exit() {
}

func (s *ShutdownScreen) Name() string {
	return "Shutdown"
}
