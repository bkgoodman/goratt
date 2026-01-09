//go:build screen

package screens

import (
	"time"

	"goratt/video/screen"
)

// AbortedScreen displays when payment is aborted/cancelled.
type AbortedScreen struct {
	mgr       *screen.Manager
	timeoutID screen.TimerID
}

// NewAbortedScreen creates a new aborted screen.
func NewAbortedScreen() *AbortedScreen {
	return &AbortedScreen{}
}

func (s *AbortedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Clear vending session
	mgr.ClearVendingSession()

	// Auto-dismiss to idle after 5 seconds
	s.timeoutID = mgr.SetTimeout(5*time.Second, func(scr screen.Screen) {
		mgr.SwitchTo(screen.ScreenIdle)
	})
}

func (s *AbortedScreen) Update() {
	s.mgr.FillBackground(0.6, 0, 0) // Red background

	// Title
	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("Cancelled", float64(s.mgr.Height()/2)-20, 1, 1, 1)

	// Instructions
	s.mgr.SetFontSize(24)
	s.mgr.DrawCentered("Press button to continue", float64(s.mgr.Height()/2)+40, 0.9, 0.9, 0.9)

	s.mgr.Flush()
}

func (s *AbortedScreen) HandleEvent(event screen.Event) bool {
	// Any button press dismisses early
	if event.Type == screen.EventRotaryPress || event.Type == screen.EventRotaryLongPress {
		s.mgr.SwitchTo(screen.ScreenIdle)
		return true
	}
	return false
}

func (s *AbortedScreen) Exit() {
	s.timeoutID = 0
}

func (s *AbortedScreen) Name() string {
	return "Aborted"
}
