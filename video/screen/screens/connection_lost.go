//go:build screen

package screens

import "goratt/video/screen"

// ConnectionLostScreen displays the connection lost state.
type ConnectionLostScreen struct {
	mgr *screen.Manager
}

// NewConnectionLostScreen creates a new connection lost screen.
func NewConnectionLostScreen() *ConnectionLostScreen {
	return &ConnectionLostScreen{}
}

func (s *ConnectionLostScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}

func (s *ConnectionLostScreen) Update() {
	s.mgr.FillBackground(0.5, 0.3, 0) // Orange-ish
	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("Connection Lost", float64(s.mgr.Height()/2), 1, 1, 1)
	s.mgr.Flush()
}

func (s *ConnectionLostScreen) HandleEvent(event screen.Event) bool {
	return false
}

func (s *ConnectionLostScreen) Exit() {
}

func (s *ConnectionLostScreen) Name() string {
	return "ConnectionLost"
}
