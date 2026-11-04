//go:build screen

package vendingscreens

import (
	"time"

	"goratt/cmd/vending/assets"
	"goratt/lib/video/screen"
)

// ApiErrorScreen displays a communication or API error.
type ApiErrorScreen struct {
	mgr       *screen.Manager
	timeoutID screen.TimerID
}

// NewApiErrorScreen creates a new API error screen.
func NewApiErrorScreen() *ApiErrorScreen {
	return &ApiErrorScreen{}
}

func (s *ApiErrorScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Clear session
	mgr.ClearVendingSession()

	// Auto-dismiss after 5 seconds
	s.timeoutID = mgr.SetTimeout(5*time.Second, func(scr screen.Screen) {
		mgr.SwitchTo(screen.ScreenIdle)
	})

	// Play error audio
	s.mgr.PlayAudioBytes(assets.Audio_error)
}

func (s *ApiErrorScreen) Update() {
	// Clear entire framebuffer first
	s.mgr.FillBackground(0.6, 0.4, 0) // Orange-ish background

	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("ERROR", float64(s.mgr.Height()/2)-40, 1, 1, 1)

	s.mgr.SetFontSize(32)
	s.mgr.DrawCentered("Communication Error", float64(s.mgr.Height()/2)+30, 0.9, 0.9, 0.9)

	s.mgr.Flush()
}

func (s *ApiErrorScreen) HandleEvent(event screen.Event) bool {
	// Any button press dismisses
	if event.Type == screen.EventRotaryPress || event.Type == screen.EventRotaryLongPress {
		s.mgr.SwitchTo(screen.ScreenIdle)
		return true
	}
	return false
}

func (s *ApiErrorScreen) Exit() {
	s.timeoutID = 0
}

func (s *ApiErrorScreen) Name() string {
	return "ApiError"
}
