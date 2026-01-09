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

	// Start 10 second timeout
	s.timeoutID = mgr.SetTimeout(10*time.Second, func(scr screen.Screen) {
		// Timeout - abort
		mgr.SwitchTo(screen.ScreenAborted)
	})
}

func (s *ConfirmScreen) Update() {
	s.mgr.FillBackground(0, 0.6, 0) // Green background

	// Title
	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("Confirm Payment", float64(s.mgr.Height()/2)-80, 1, 1, 1)

	// Display member name
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DrawCentered(displayName, float64(s.mgr.Height()/2)-30, 0.9, 0.9, 0.9)
	}

	// Display amount
	s.mgr.SetFontSize(64)
	amountStr := fmt.Sprintf("$%.2f", s.amount)
	s.mgr.DrawCentered(amountStr, float64(s.mgr.Height()/2)+30, 1, 1, 0)

	// Instructions
	s.mgr.SetFontSize(24)
	s.mgr.DrawCentered("Press to complete", float64(s.mgr.Height()/2)+90, 0.9, 0.9, 0.9)
	s.mgr.DrawCentered("Hold to cancel", float64(s.mgr.Height()/2)+120, 0.9, 0.9, 0.9)

	s.mgr.Flush()
}

func (s *ConfirmScreen) HandleEvent(event screen.Event) bool {
	switch event.Type {
	case screen.EventRotaryPress:
		// Short press - complete payment and return to idle
		s.mgr.ClearVendingSession()
		s.mgr.SwitchTo(screen.ScreenIdle)
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
