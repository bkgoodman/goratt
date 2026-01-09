//go:build screen

package screens

import (
	"fmt"
	"time"

	"goratt/video/screen"
)

// InsufficientFundsScreen allows user to add money when balance is too low.
type InsufficientFundsScreen struct {
	mgr            *screen.Manager
	member         string
	nickname       string
	purchaseAmount float64
	balance        float64
	addAmount      float64
	timeoutID      screen.TimerID
	minAdd         float64
	maxAdd         float64
	step           float64
	timeoutPeriod  time.Duration

	// Batched UI updates
	updateTimerID  screen.TimerID
	pendingUpdate  bool
	updateInterval time.Duration

	// Partial update area
	updateX      int
	updateY      int
	updateWidth  int
	updateHeight int
}

// NewInsufficientFundsScreen creates a new insufficient funds screen.
func NewInsufficientFundsScreen() *InsufficientFundsScreen {
	return &InsufficientFundsScreen{
		minAdd:         1.00,
		maxAdd:         10.00,
		step:           1.00,
		timeoutPeriod:  30 * time.Second,
		updateInterval: 50 * time.Millisecond,
	}
}

func (s *InsufficientFundsScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Get session info
	s.member, s.nickname, s.purchaseAmount = mgr.GetVendingSession()
	s.balance = mgr.GetVendingBalance()
	s.addAmount = mgr.GetVendingAddAmount()

	// Default to $5 if not set
	if s.addAmount == 0 {
		s.addAmount = 5.00
		mgr.SetVendingAddAmount(s.addAmount)
	}

	// Calculate partial update area (center region for numbers)
	s.updateWidth = 280
	s.updateHeight = 140
	s.updateX = (mgr.Width() - s.updateWidth) / 2
	s.updateY = mgr.Height()/2 - 40

	// Reset batching state
	s.pendingUpdate = false
	s.updateTimerID = 0

	// Start timeout timer
	s.startTimeout()
}

func (s *InsufficientFundsScreen) startTimeout() {
	s.timeoutID = s.mgr.SetTimeout(s.timeoutPeriod, func(scr screen.Screen) {
		s.mgr.SwitchTo(screen.ScreenAborted)
	})
}

func (s *InsufficientFundsScreen) Update() {
	s.mgr.FillBackground(0.6, 0.3, 0) // Orange/warning background

	// Warning title
	s.mgr.SetFontSize(36)
	s.mgr.DrawCentered("Insufficient Funds", float64(s.mgr.Height()/2)-100, 1, 1, 1)

	// Display amounts
	s.drawAmounts()

	// Instructions
	s.mgr.SetFontSize(18)
	s.mgr.DrawCentered("Turn knob to adjust", float64(s.mgr.Height()/2)+110, 0.9, 0.9, 0.9)
	s.mgr.DrawCentered("Press to confirm", float64(s.mgr.Height()/2)+135, 0.9, 0.9, 0.9)

	s.mgr.Flush()
}

func (s *InsufficientFundsScreen) drawAmounts() {
	centerY := float64(s.mgr.Height() / 2)

	s.mgr.SetFontSize(20)

	// Purchase amount
	s.mgr.DrawCentered(fmt.Sprintf("Purchase: $%.2f", s.purchaseAmount), centerY-50, 1, 1, 1)

	// Current balance
	s.mgr.DrawCentered(fmt.Sprintf("Balance: $%.2f", s.balance), centerY-20, 1, 1, 1)

	// Add amount (highlighted)
	s.mgr.SetFontSize(32)
	s.mgr.DrawCentered(fmt.Sprintf("Add: $%.2f", s.addAmount), centerY+20, 1, 1, 0)

	// Remaining after transaction
	remaining := s.balance + s.addAmount - s.purchaseAmount
	s.mgr.SetFontSize(24)
	remainingColor := 0.5
	if remaining >= 0 {
		remainingColor = 1.0 // Bright if positive
	}
	s.mgr.DrawCentered(fmt.Sprintf("After: $%.2f", remaining), centerY+60, remainingColor, 1, remainingColor)
}

// updateAmountsDisplay does a partial update of just the amounts area
func (s *InsufficientFundsScreen) updateAmountsDisplay() {
	// Clear the update area with background color
	s.mgr.DC().SetRGB(0.6, 0.3, 0)
	s.mgr.DC().DrawRectangle(float64(s.updateX), float64(s.updateY), float64(s.updateWidth), float64(s.updateHeight))
	s.mgr.DC().Fill()

	// Redraw amounts
	s.drawAmounts()

	// Flush only the update area
	s.mgr.FlushRect(s.updateX, s.updateY, s.updateWidth, s.updateHeight)
}

func (s *InsufficientFundsScreen) HandleEvent(event screen.Event) bool {
	switch event.Type {
	case screen.EventRotaryTurn:
		if rotary := event.Rotary(); rotary != nil {
			// Adjust add amount immediately
			s.addAmount += float64(rotary.Delta) * s.step

			// Clamp to min/max
			if s.addAmount < s.minAdd {
				s.addAmount = s.minAdd
			}
			if s.addAmount > s.maxAdd {
				s.addAmount = s.maxAdd
			}

			// Update session
			s.mgr.SetVendingAddAmount(s.addAmount)

			// Schedule batched UI update if not already pending
			if !s.pendingUpdate {
				s.pendingUpdate = true
				s.updateTimerID = s.mgr.SetTimeout(s.updateInterval, func(scr screen.Screen) {
					if s.updateTimerID != 0 {
						s.pendingUpdate = false
						s.updateTimerID = 0
						s.updateAmountsDisplay()
					}
				})
			}

			// Reset timeout
			if s.timeoutID != 0 {
				s.mgr.ClearTimeout(s.timeoutID)
			}
			s.startTimeout()

			return true
		}

	case screen.EventRotaryPress:
		// Short press - proceed to confirm with added funds
		s.mgr.SwitchTo(screen.ScreenConfirm)
		return true

	case screen.EventRotaryLongPress:
		// Long press - abort
		s.mgr.SwitchTo(screen.ScreenAborted)
		return true
	}
	return false
}

func (s *InsufficientFundsScreen) Exit() {
	s.timeoutID = 0
	s.updateTimerID = 0
	s.pendingUpdate = false
}

func (s *InsufficientFundsScreen) Name() string {
	return "InsufficientFunds"
}
