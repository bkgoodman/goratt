//go:build screen

package vendingscreens

import (
	"fmt"
	"time"

	"goratt/lib/video/screen"
)

// PaymentFailedScreen displays payment failure.
type PaymentFailedScreen struct {
	mgr       *screen.Manager
	timeoutID screen.TimerID
	amount    float64
	addAmount float64
}

// NewPaymentFailedScreen creates a new payment failed screen.
func NewPaymentFailedScreen() *PaymentFailedScreen {
	return &PaymentFailedScreen{}
}

func (s *PaymentFailedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Get amounts from session
	_, _, s.amount = mgr.GetVendingSession()
	s.addAmount = mgr.GetVendingAddAmount()

	// Clear session after failed payment
	mgr.ClearVendingSession()

	// Force a full framebuffer clear to prevent any artifacts from previous screen
	mgr.FillBackground(0.6, 0, 0)
	mgr.Flush()

	// Auto-dismiss after 10 seconds
	s.timeoutID = mgr.SetTimeout(10*time.Second, func(scr screen.Screen) {
		mgr.SwitchTo(screen.ScreenIdle)
	})

	// Play purchase audio
	s.mgr.PlayAudio("error_16.pcm")
}

func (s *PaymentFailedScreen) Update() {
	// Clear entire framebuffer first
	s.mgr.FillBackground(0.6, 0, 0) // Red background

	// Failure title (no special character)
	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("FAILED", float64(s.mgr.Height()/2)-60, 1, 1, 1)

	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("Payment Failed", float64(s.mgr.Height()/2)-10, 1, 1, 1)

	// Show transaction details
	s.mgr.SetFontSize(24)
	if s.addAmount > 0 {
		s.mgr.DrawCentered(fmt.Sprintf("Amount: $%.2f", s.amount), float64(s.mgr.Height()/2)+30, 0.9, 0.9, 0.9)
		s.mgr.DrawCentered(fmt.Sprintf("Add: $%.2f", s.addAmount), float64(s.mgr.Height()/2)+60, 0.9, 0.9, 0.9)
	} else {
		s.mgr.DrawCentered(fmt.Sprintf("Amount: $%.2f", s.amount), float64(s.mgr.Height()/2)+30, 0.9, 0.9, 0.9)
	}

	// Instructions
	s.mgr.SetFontSize(20)
	s.mgr.DrawCentered("Press button to continue", float64(s.mgr.Height()/2)+100, 0.8, 0.8, 0.8)

	s.mgr.Flush()
}

func (s *PaymentFailedScreen) HandleEvent(event screen.Event) bool {
	// Any button press dismisses
	if event.Type == screen.EventRotaryPress || event.Type == screen.EventRotaryLongPress {
		s.mgr.SwitchTo(screen.ScreenIdle)
		return true
	}
	return false
}

func (s *PaymentFailedScreen) Exit() {
	s.timeoutID = 0
}

func (s *PaymentFailedScreen) Name() string {
	return "PaymentFailed"
}
