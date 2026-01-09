//go:build screen

package screens

import (
	"fmt"
	"time"

	"goratt/video/screen"
)

// SelectAmountScreen allows user to select payment amount with rotary encoder.
type SelectAmountScreen struct {
	mgr           *screen.Manager
	amount        float64
	balance       float64
	timeoutID     screen.TimerID
	member        string
	nickname      string
	minAmount     float64
	maxAmount     float64
	step          float64
	timeoutPeriod time.Duration

	// Partial update area for amount display
	amountX      int
	amountY      int
	amountWidth  int
	amountHeight int

	// Batched UI updates to avoid blocking rotary encoder
	updateTimerID  screen.TimerID
	pendingUpdate  bool
	updateInterval time.Duration
}

// NewSelectAmountScreen creates a new select amount screen.
func NewSelectAmountScreen() *SelectAmountScreen {
	return &SelectAmountScreen{
		minAmount:      0.75,
		maxAmount:      5.00,
		step:           0.25,
		timeoutPeriod:  30 * time.Second,
		updateInterval: 50 * time.Millisecond, // Batch updates every 50ms
	}
}

func (s *SelectAmountScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Get member info from vending session
	s.member, s.nickname, s.amount = mgr.GetVendingSession()
	s.balance = mgr.GetVendingBalance()

	// Default to $1.00 if not set
	if s.amount == 0 {
		s.amount = 1.00
		mgr.SetVendingSession(s.member, s.nickname, s.amount)
	}

	// Calculate partial update area for amount (centered, ~250px wide x 90px tall)
	// Amount text is at height/2 + 40, so center the box around that
	s.amountWidth = 250
	s.amountHeight = 90
	s.amountX = (mgr.Width() - s.amountWidth) / 2
	s.amountY = mgr.Height()/2 - 5 // Moved up to avoid overlapping instructions

	// Reset batching state
	s.pendingUpdate = false
	s.updateTimerID = 0

	// Start timeout timer
	s.startTimeout()
}

func (s *SelectAmountScreen) startTimeout() {
	s.timeoutID = s.mgr.SetTimeout(s.timeoutPeriod, func(scr screen.Screen) {
		// Timeout - go to aborted screen
		s.mgr.SwitchTo(screen.ScreenAborted)
	})
}

func (s *SelectAmountScreen) Update() {
	s.mgr.FillBackground(0, 0.4, 0.6) // Blue background

	// Title
	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("Select Amount", float64(s.mgr.Height()/2)-90, 1, 1, 1)

	// Display member name
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(28)
		s.mgr.DrawCentered(displayName, float64(s.mgr.Height()/2)-55, 0.9, 0.9, 0.9)
	}

	// Display current balance
	s.mgr.SetFontSize(20)
	s.mgr.DrawCentered(fmt.Sprintf("Balance: $%.2f", s.balance), float64(s.mgr.Height()/2)-25, 0.8, 0.8, 0.8)

	// Display amount (large)
	s.mgr.SetFontSize(72)
	amountStr := fmt.Sprintf("$%.2f", s.amount)
	s.mgr.DrawCentered(amountStr, float64(s.mgr.Height()/2)+40, 1, 1, 0)

	// Instructions
	s.mgr.SetFontSize(20)
	s.mgr.DrawCentered("Turn knob to adjust", float64(s.mgr.Height()/2)+100, 0.9, 0.9, 0.9)
	s.mgr.DrawCentered("Press to confirm", float64(s.mgr.Height()/2)+130, 0.9, 0.9, 0.9)

	s.mgr.Flush()
}

// updateAmountDisplay does a partial update of just the amount area
func (s *SelectAmountScreen) updateAmountDisplay() {
	// Clear the amount area with background color
	s.mgr.DC().SetRGB(0, 0.4, 0.6)
	s.mgr.DC().DrawRectangle(float64(s.amountX), float64(s.amountY), float64(s.amountWidth), float64(s.amountHeight))
	s.mgr.DC().Fill()

	// Draw the amount text
	s.mgr.SetFontSize(72)
	amountStr := fmt.Sprintf("$%.2f", s.amount)
	s.mgr.DrawCentered(amountStr, float64(s.mgr.Height()/2)+40, 1, 1, 0)

	// Flush only the amount area
	s.mgr.FlushRect(s.amountX, s.amountY, s.amountWidth, s.amountHeight)
}

func (s *SelectAmountScreen) HandleEvent(event screen.Event) bool {
	switch event.Type {
	case screen.EventRotaryTurn:
		if rotary := event.Rotary(); rotary != nil {
			// Adjust amount immediately (fast, no UI blocking)
			s.amount += float64(rotary.Delta) * s.step

			// Clamp to min/max
			if s.amount < s.minAmount {
				s.amount = s.minAmount
			}
			if s.amount > s.maxAmount {
				s.amount = s.maxAmount
			}

			// Update session
			s.mgr.SetVendingSession(s.member, s.nickname, s.amount)

			// Schedule batched UI update if not already pending
			if !s.pendingUpdate {
				s.pendingUpdate = true
				s.updateTimerID = s.mgr.SetTimeout(s.updateInterval, func(scr screen.Screen) {
					// Check if we're still the active screen
					if s.updateTimerID != 0 {
						s.pendingUpdate = false
						s.updateTimerID = 0
						s.updateAmountDisplay()
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
		// Short press - go to confirm screen
		s.mgr.SwitchTo(screen.ScreenConfirm)
		return true

	case screen.EventRotaryLongPress:
		// Long press - abort
		s.mgr.SwitchTo(screen.ScreenAborted)
		return true
	}
	return false
}

func (s *SelectAmountScreen) Exit() {
	s.timeoutID = 0
	s.updateTimerID = 0
	s.pendingUpdate = false
}

func (s *SelectAmountScreen) Name() string {
	return "SelectAmount"
}
