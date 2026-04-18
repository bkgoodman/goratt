//go:build screen

package vendingscreens

import (
	"fmt"
	"strings"
	"time"

	"goratt/cmd/vending/assets"
	"goratt/lib/video/screen"
	"goratt/lib/video/screen/screens"
)

// ConfirmScreen displays payment confirmation and waits for final confirmation.
type ConfirmScreen struct {
	mgr           *screen.Manager
	member        string
	amount        float64
	balance       float64
	addAmount     float64
	timeoutID     screen.TimerID
	cancelOverlay *screens.CancelOverlay
}

// NewConfirmScreen creates a new confirm screen.
func NewConfirmScreen() *ConfirmScreen {
	return &ConfirmScreen{}
}

func (s *ConfirmScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Get session info
	s.member, _, s.amount = mgr.GetVendingSession()
	s.balance = mgr.GetVendingBalance()
	s.addAmount = mgr.GetVendingAddAmount()

	// Initialize cancel overlay
	config := screens.DefaultCancelOverlayConfig(mgr)
	s.cancelOverlay = screens.NewCancelOverlay(mgr, config)

	// Check if balance is sufficient
	totalBalance := s.balance + s.addAmount
	if s.amount > totalBalance {
		// Insufficient funds - redirect to add money screen
		// Don't set timeout since we're immediately leaving this screen
		mgr.SwitchTo(screen.ScreenInsufficientFunds)
		return
	}

	// Start inactivity timeout
	s.resetTimeout()

	// Play purchase audio
	s.mgr.PlayAudioBytes(assets.Audio_confirm)
}

func (s *ConfirmScreen) resetTimeout() {
	if s.timeoutID != 0 {
		s.mgr.ClearTimeout(s.timeoutID)
	}
	s.timeoutID = s.mgr.SetTimeout(8*time.Second, func(scr screen.Screen) {
		// Inactivity timeout - start visual cancel countdown
		s.cancelOverlay.Start(screens.CancelModeTimeout, func() {
			// Cancel completed
			s.mgr.SwitchTo(screen.ScreenAborted)
		}, func() {
			// Cancel aborted - restart inactivity timeout
			s.resetTimeout()
		})
	})
}

func (s *ConfirmScreen) Update() {
	s.mgr.FillBackground(0, 0.6, 0) // Green background

	// Title
	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("Confirm Payment", float64(s.mgr.Height()/2)-90, 1, 1, 1)

	// Display member name
	displayName := strings.ReplaceAll(s.member, ".", " ")
	if displayName != "" {
		s.mgr.SetFontSize(28)
		s.mgr.DrawCentered(displayName, float64(s.mgr.Height()/2)-155, 0.9, 0.9, 0.9)
	}

	centerY := float64(s.mgr.Height() / 2)

	// If adding funds, show both amounts separately
	if s.addAmount > 0 {
		fee := 0.0
		if s.addAmount < 5.0 {
			fee = 0.30
		}

		s.mgr.SetFontSize(24)
		s.mgr.DrawCentered(fmt.Sprintf("Purchase: $%.2f", s.amount), centerY-20, 1, 1, 1)
		s.mgr.DrawCentered(fmt.Sprintf("Adding: $%.2f", s.addAmount), centerY+10, 1, 1, 0)
		
		if fee > 0 {
			s.mgr.SetFontSize(16)
			s.mgr.DrawCentered(fmt.Sprintf("+ $%.2f Service Fee", fee), centerY+30, 1, 0.8, 0.8)
		}

		// Show total and remaining
		totalBalance := s.balance + s.addAmount
		remaining := totalBalance - s.amount
		totalCharge := s.addAmount + fee
		
		s.mgr.SetFontSize(20)
		s.mgr.DrawCentered(fmt.Sprintf("New Balance: $%.2f", remaining), centerY+55, 0.8, 1, 0.8)
		s.mgr.DrawCentered(fmt.Sprintf("I consent to charge $%.2f", totalCharge), float64(s.mgr.Height()/2)+155, 0.9, 0.9, 0.9)
	} else {
		// Just purchase, no add
		s.mgr.SetFontSize(64)
		amountStr := fmt.Sprintf("$%.2f", s.amount)
		s.mgr.DrawCentered(amountStr, centerY+10, 1, 1, 0)

		// Display remaining balance
		remaining := s.balance - s.amount
		s.mgr.SetFontSize(24)
		s.mgr.DrawCentered(fmt.Sprintf("Remaining: $%.2f", remaining), centerY+60, 0.9, 0.9, 0.9)
	}

	// Instructions
	s.mgr.SetFontSize(20)
	s.mgr.DrawCentered("Press to complete", float64(s.mgr.Height()/2)+95, 0.9, 0.9, 0.9)
	s.mgr.DrawCentered("Hold to cancel", float64(s.mgr.Height()/2)+120, 0.9, 0.9, 0.9)

	// Draw cancel overlay if active
	s.cancelOverlay.Draw()

	s.mgr.Flush()
}

func (s *ConfirmScreen) HandleEvent(event screen.Event) bool {
	// If cancel overlay is active, let it handle interaction
	if s.cancelOverlay.HandleEvent(event) {
		return true
	}

	switch event.Type {
	case screen.EventRotaryTurn:
		s.resetTimeout()
		return true

	case screen.EventRotaryPress:
		// Short press - go to processing screen
		s.mgr.SwitchTo(screen.ScreenProcessing)
		return true

	case screen.EventRotaryLongPress:
		// Long press - start cancel sequence
		s.cancelOverlay.Start(screens.CancelModeHold, func() {
			s.mgr.SwitchTo(screen.ScreenAborted)
		}, nil)
		return true
	}
	return false
}

func (s *ConfirmScreen) Exit() {
	s.timeoutID = 0
	if s.cancelOverlay != nil {
		s.cancelOverlay.Reset()
	}
}

func (s *ConfirmScreen) Name() string {
	return "Confirm"
}
