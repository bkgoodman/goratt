//go:build screen

package darkroomscreens

import (
	"goratt/lib/video/screen"
)

type SafeLightScreen struct {
	mgr *screen.Manager
}

func NewSafeLightScreen() *SafeLightScreen {
	return &SafeLightScreen{}
}

func (s *SafeLightScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}

func (s *SafeLightScreen) Update() {
	s.mgr.FillBackground(0.4, 0, 0) // Very deep red

	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("SAFE-LIGHT ON", y, 1, 0, 0)

	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("DO NOT ENTER", y+80, 1, 0, 0)

	s.mgr.Flush()
}

func (s *SafeLightScreen) HandleEvent(event screen.Event) bool {
	if event.Type == screen.EventRotaryPress {
		s.mgr.SwitchTo(screen.ScreenIdle)
		return true
	}
	return false
}

func (s *SafeLightScreen) Exit() {
}

func (s *SafeLightScreen) Name() string {
	return "SafeLight"
}
