//go:build screen

package screens

import (
	"fmt"
	"time"

	"goratt/video/screen"
)

// SuccessScreen displays payment success.
type SuccessScreen struct {
	mgr       *screen.Manager
	timeoutID screen.TimerID
	amount    float64
	addAmount float64
}

// NewSuccessScreen creates a new success screen.
func NewSuccessScreen() *SuccessScreen {
	return &SuccessScreen{}
}

func (s *SuccessScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Get amounts from session
	_, _, s.amount = mgr.GetVendingSession()
	s.addAmount = mgr.GetVendingAddAmount()

	// Clear session after successful payment
	mgr.ClearVendingSession()

	// Auto-dismiss after 10 seconds
	s.timeoutID = mgr.SetTimeout(10*time.Second, func(scr screen.Screen) {
		mgr.SwitchTo(screen.ScreenIdle)
	})
}

func (s *SuccessScreen) Update() {
	s.mgr.FillBackground(0, 0.6, 0) // Green background

	// Success icon/title
	s.mgr.SetFontSize(72)
	s.mgr.DrawCentered("✓", float64(s.mgr.Height()/2)-60, 1, 1, 1)

	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("Success!", float64(s.mgr.Height()/2), 1, 1, 1)

	// Show transaction details
	s.mgr.SetFontSize(24)
	if s.addAmount > 0 {
		s.mgr.DrawCentered(fmt.Sprintf("Paid: $%.2f", s.amount), float64(s.mgr.Height()/2)+50, 0.9, 0.9, 0.9)
		s.mgr.DrawCentered(fmt.Sprintf("Added: $%.2f", s.addAmount), float64(s.mgr.Height()/2)+80, 0.9, 0.9, 0.9)
	} else {
		s.mgr.DrawCentered(fmt.Sprintf("Paid: $%.2f", s.amount), float64(s.mgr.Height()/2)+50, 0.9, 0.9, 0.9)
	}

	// Instructions
	s.mgr.SetFontSize(20)
	s.mgr.DrawCentered("Press button to continue", float64(s.mgr.Height()/2)+120, 0.8, 0.8, 0.8)

	s.mgr.Flush()
}

func (s *SuccessScreen) HandleEvent(event screen.Event) bool {
	// Any button press dismisses
	if event.Type == screen.EventRotaryPress || event.Type == screen.EventRotaryLongPress {
		s.mgr.SwitchTo(screen.ScreenIdle)
		return true
	}
	return false
}

func (s *SuccessScreen) Exit() {
	s.timeoutID = 0
}

func (s *SuccessScreen) Name() string {
	return "Success"
}
