//go:build screen

package screens

import (
	"fmt"
	"time"

	"goratt/video/screen"
)

// IdleScreen displays the ready/idle state.
type IdleScreen struct {
	mgr         *screen.Manager
	counter     int
	showCounter bool
	timerID     screen.TimerID

	// Counter display area for partial updates
	counterY      int
	counterHeight int
}

// NewIdleScreen creates a new idle screen.
func NewIdleScreen() *IdleScreen {
	return &IdleScreen{}
}

func (s *IdleScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	s.counter = 0
	s.showCounter = false
	s.timerID = 0

	// Calculate counter area (bottom half of screen, 60px tall)
	s.counterY = mgr.Height()/2 + 10
	s.counterHeight = 60
}

func (s *IdleScreen) Update() {
	s.mgr.FillBackground(0, 0.5, 0) // Green background
	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("Ready", float64(s.mgr.Height()/2)-30, 1, 1, 1)

	// Show debug counter if active
	if s.showCounter {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(fmt.Sprintf("%d", s.counter), float64(s.mgr.Height()/2)+40, 1, 1, 0)
	}

	s.mgr.Flush()
}

// updateCounter does a partial update of just the counter area
func (s *IdleScreen) updateCounter() {
	// Clear the counter area with background color
	s.mgr.FillRect(0, s.counterY, s.mgr.Width(), s.counterHeight, 0, 0.5, 0)

	// Draw counter if visible
	if s.showCounter {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(fmt.Sprintf("%d", s.counter), float64(s.mgr.Height()/2)+40, 1, 1, 0)
	}

	// Flush only the counter area
	s.mgr.FlushRect(0, s.counterY, s.mgr.Width(), s.counterHeight)
}

func (s *IdleScreen) HandleEvent(event screen.Event) bool {
	switch event.Type {
	case screen.EventRotaryTurn:
		if rotary := event.Rotary(); rotary != nil {
			s.counter += rotary.Delta
			s.showCounter = true
			s.resetTimeout()
			s.updateCounter()
			return true
		}
	case screen.EventRotaryPress:
		if s.showCounter {
			s.showCounter = false
			s.counter = 0
			if s.timerID != 0 {
				s.mgr.ClearTimeout(s.timerID)
				s.timerID = 0
			}
			s.updateCounter()
			return true
		}
	}
	return false
}

func (s *IdleScreen) resetTimeout() {
	// Clear existing timer
	if s.timerID != 0 {
		s.mgr.ClearTimeout(s.timerID)
	}
	// Set new 10 second timeout to hide counter
	s.timerID = s.mgr.SetTimeout(10*time.Second, func(scr screen.Screen) {
		s.showCounter = false
		s.counter = 0
		s.timerID = 0
		s.updateCounter()
	})
}

func (s *IdleScreen) Exit() {
	s.showCounter = false
	s.counter = 0
	s.timerID = 0
}

func (s *IdleScreen) Name() string {
	return "Idle"
}
