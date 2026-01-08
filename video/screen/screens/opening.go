//go:build screen

package screens

import (
	"goratt/video/screen"
	"time"
)

// OpeningScreen displays the door opening state.
type OpeningScreen struct {
	mgr      *screen.Manager
	member   string
	nickname string
	warning  string

	// Spinner animation
	spinnerFrame   int
	spinnerTimerID screen.TimerID
	spinnerX       int
	spinnerY       int
}

// NewOpeningScreen creates a new opening screen.
func NewOpeningScreen() *OpeningScreen {
	return &OpeningScreen{}
}

// SetInfo sets the member info to display.
func (s *OpeningScreen) SetInfo(member, nickname, warning string) {
	s.member = member
	s.nickname = nickname
	s.warning = warning
}

func (s *OpeningScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Spinner position - centered below text content
	s.spinnerX = (mgr.Width() - spinnerSize) / 2
	s.spinnerY = mgr.Height() - 60 // Near bottom of screen

	// Start spinner animation
	s.startSpinnerAnimation()
}

func (s *OpeningScreen) Update() {
	s.mgr.FillBackground(0.7, 0.7, 0) // Yellow

	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("Opening...", y, 0, 0, 0)

	// Display name
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(displayName, y+70, 0, 0, 0)
	}

	// Display warning if present
	if s.warning != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DC().SetRGB(0.7, 0, 0) // Red warning text on yellow background
		s.mgr.DC().DrawStringAnchored(s.warning, float64(s.mgr.Width()/2), y+130, 0.5, 0.5)
	}

	// Draw spinner
	s.mgr.DC().DrawImage(spinnerFrames[s.spinnerFrame], s.spinnerX, s.spinnerY)

	s.mgr.Flush()
}

// startSpinnerAnimation starts the 100ms timer for spinner animation
func (s *OpeningScreen) startSpinnerAnimation() {
	s.spinnerTimerID = s.mgr.SetTimeout(100*time.Millisecond, func(scr screen.Screen) {
		// Only continue if timer wasn't cleared (screen still active)
		if s.spinnerTimerID == 0 {
			return
		}
		s.spinnerFrame = (s.spinnerFrame + 1) % len(spinnerFrames)
		s.updateSpinner()
		s.startSpinnerAnimation() // Schedule next frame
	})
}

// updateSpinner does a partial update of just the spinner area
func (s *OpeningScreen) updateSpinner() {
	s.mgr.DC().DrawImage(spinnerFrames[s.spinnerFrame], s.spinnerX, s.spinnerY)
	s.mgr.FlushRect(s.spinnerX, s.spinnerY, spinnerSize, spinnerSize)
}

func (s *OpeningScreen) HandleEvent(event screen.Event) bool {
	return false
}

func (s *OpeningScreen) Exit() {
	// Mark spinner as stopped (timer is cleared by manager)
	s.spinnerTimerID = 0
	s.member = ""
	s.nickname = ""
	s.warning = ""
}

func (s *OpeningScreen) Name() string {
	return "Opening"
}
