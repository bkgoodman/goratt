//go:build screen

package vendingscreens

import (
	"fmt"
	"time"

	"goratt/lib/video/screen"
	"goratt/cmd/vending/assets"
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

	// Play purchase audio
	s.mgr.PlayAudioBytes(assets.Audio_complete)
}

func (s *SuccessScreen) Update() {
	// Clear entire framebuffer first
	s.mgr.FillBackground(0, 0.6, 0) // Green background

	// Success title (larger, no special character)
	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("SUCCESS", float64(s.mgr.Height()/2)-60, 1, 1, 1)

	// Success message
	s.mgr.SetFontSize(36)
	s.mgr.DrawCentered("Payment Complete", float64(s.mgr.Height()/2)-10, 1, 1, 1)

	// Show transaction details
	s.mgr.SetFontSize(24)
	if s.addAmount > 0 {
		s.mgr.DrawCentered(fmt.Sprintf("Paid: $%.2f", s.amount), float64(s.mgr.Height()/2)+30, 0.9, 0.9, 0.9)
		s.mgr.DrawCentered(fmt.Sprintf("Added: $%.2f", s.addAmount), float64(s.mgr.Height()/2)+60, 0.9, 0.9, 0.9)
	} else {
		s.mgr.DrawCentered(fmt.Sprintf("Paid: $%.2f", s.amount), float64(s.mgr.Height()/2)+30, 0.9, 0.9, 0.9)
	}

	// Instructions
	s.mgr.SetFontSize(20)
	s.mgr.DrawCentered("Press button to continue", float64(s.mgr.Height()/2)+100, 0.8, 0.8, 0.8)

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
