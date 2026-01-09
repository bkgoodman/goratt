//go:build screen

package screens

import (
	"fmt"
	"time"

	"goratt/video/screen"
)

// ConfirmScreen displays payment confirmation and waits for final confirmation.
type ConfirmScreen struct {
	mgr       *screen.Manager
	member    string
	nickname  string
	amount    float64
	balance   float64
	addAmount float64
	timeoutID screen.TimerID
}

// NewConfirmScreen creates a new confirm screen.
func NewConfirmScreen() *ConfirmScreen {
	return &ConfirmScreen{}
}

func (s *ConfirmScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Get session info
	s.member, s.nickname, s.amount = mgr.GetVendingSession()
	s.balance = mgr.GetVendingBalance()
	s.addAmount = mgr.GetVendingAddAmount()

	// Check if balance is sufficient
	totalBalance := s.balance + s.addAmount
	if s.amount > totalBalance {
		// Insufficient funds - redirect to add money screen
		// Don't set timeout since we're immediately leaving this screen
		mgr.SwitchTo(screen.ScreenInsufficientFunds)
		return
	}

	// Only start timeout if we're staying on this screen
	s.timeoutID = mgr.SetTimeout(10*time.Second, func(scr screen.Screen) {
		// Timeout - abort
		mgr.SwitchTo(screen.ScreenAborted)
	})
}

func (s *ConfirmScreen) Update() {
	s.mgr.FillBackground(0, 0.6, 0) // Green background

	// Title
	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("Confirm Payment", float64(s.mgr.Height()/2)-90, 1, 1, 1)

	// Display member name
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(28)
		s.mgr.DrawCentered(displayName, float64(s.mgr.Height()/2)-50, 0.9, 0.9, 0.9)
	}

	centerY := float64(s.mgr.Height() / 2)

	// If adding funds, show both amounts separately
	if s.addAmount > 0 {
		s.mgr.SetFontSize(24)
		s.mgr.DrawCentered(fmt.Sprintf("Purchase: $%.2f", s.amount), centerY-10, 1, 1, 1)
		s.mgr.DrawCentered(fmt.Sprintf("Adding: $%.2f", s.addAmount), centerY+20, 1, 1, 0)

		// Show total and remaining
		totalBalance := s.balance + s.addAmount
		remaining := totalBalance - s.amount
		s.mgr.SetFontSize(20)
		s.mgr.DrawCentered(fmt.Sprintf("New Balance: $%.2f", remaining), centerY+55, 0.8, 1, 0.8)
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

	s.mgr.Flush()
}

func (s *ConfirmScreen) HandleEvent(event screen.Event) bool {
	switch event.Type {
	case screen.EventRotaryPress:
		// Short press - go to processing screen
		s.mgr.SwitchTo(screen.ScreenProcessing)
		return true

	case screen.EventRotaryLongPress:
		// Long press - abort
		s.mgr.SwitchTo(screen.ScreenAborted)
		return true
	}
	return false
}

func (s *ConfirmScreen) Exit() {
	s.timeoutID = 0
}

func (s *ConfirmScreen) Name() string {
	return "Confirm"
}
