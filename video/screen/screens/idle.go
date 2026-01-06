//go:build screen

package screens

import "goratt/video/screen"

// IdleScreen displays the ready/idle state.
type IdleScreen struct {
	mgr *screen.Manager
}

// NewIdleScreen creates a new idle screen.
func NewIdleScreen() *IdleScreen {
	return &IdleScreen{}
}

func (s *IdleScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}

func (s *IdleScreen) Update() {
	s.mgr.FillBackground(0, 0.5, 0) // Green background
	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("Ready", float64(s.mgr.Height()/2), 1, 1, 1)
	s.mgr.Flush()
}

func (s *IdleScreen) HandleEvent(event screen.Event) bool {
	// Idle screen doesn't handle events directly;
	// the main app handles RFID and transitions to other screens
	return false
}

func (s *IdleScreen) Exit() {
	// Nothing to clean up
}

func (s *IdleScreen) Name() string {
	return "Idle"
}
