//go:build screen

package screens

import (
	"math/rand"
	"time"

	"goratt/video/screen"
)

// ProcessingScreen shows a spinner while processing payment.
type ProcessingScreen struct {
	mgr            *screen.Manager
	timeoutID      screen.TimerID
	spinnerTimerID screen.TimerID
	spinnerFrame   int
	spinnerX       int
	spinnerY       int
}

// NewProcessingScreen creates a new processing screen.
func NewProcessingScreen() *ProcessingScreen {
	return &ProcessingScreen{}
}

func (s *ProcessingScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	s.spinnerFrame = 0

	// Position spinner below text
	s.spinnerX = (mgr.Width() - spinnerSize) / 2
	s.spinnerY = mgr.Height()/2 + 20

	// Start spinner animation
	s.startSpinnerAnimation()

	// After 5 seconds, randomly go to Success or Failed
	s.timeoutID = mgr.SetTimeout(5*time.Second, func(scr screen.Screen) {
		// Random success/failure for now
		if rand.Float32() < 0.5 {
			mgr.SwitchTo(screen.ScreenSuccess)
		} else {
			mgr.SwitchTo(screen.ScreenPaymentFailed)
		}
	})
}

func (s *ProcessingScreen) startSpinnerAnimation() {
	s.spinnerTimerID = s.mgr.SetTimeout(100*time.Millisecond, func(scr screen.Screen) {
		if s.spinnerTimerID == 0 {
			return
		}
		s.spinnerFrame = (s.spinnerFrame + 1) % len(spinnerFrames)
		s.updateSpinner()
		s.startSpinnerAnimation()
	})
}

func (s *ProcessingScreen) Update() {
	s.mgr.FillBackground(0, 0.4, 0.6) // Blue background

	// Title
	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("Processing", float64(s.mgr.Height()/2)-40, 1, 1, 1)

	// Subtitle
	s.mgr.SetFontSize(24)
	s.mgr.DrawCentered("Please wait...", float64(s.mgr.Height()/2)+5, 0.9, 0.9, 0.9)

	// Draw spinner
	s.drawSpinner()

	s.mgr.Flush()
}

func (s *ProcessingScreen) drawSpinner() {
	if s.spinnerFrame < len(spinnerFrames) {
		frame := spinnerFrames[s.spinnerFrame]
		s.mgr.DC().DrawImage(frame, s.spinnerX, s.spinnerY)
	}
}

func (s *ProcessingScreen) updateSpinner() {
	// Clear spinner area
	s.mgr.DC().SetRGB(0, 0.4, 0.6)
	s.mgr.DC().DrawRectangle(float64(s.spinnerX), float64(s.spinnerY), float64(spinnerSize), float64(spinnerSize))
	s.mgr.DC().Fill()

	// Draw new frame
	s.drawSpinner()

	// Flush only spinner area
	s.mgr.FlushRect(s.spinnerX, s.spinnerY, spinnerSize, spinnerSize)
}

func (s *ProcessingScreen) HandleEvent(event screen.Event) bool {
	// No user interaction during processing
	return false
}

func (s *ProcessingScreen) Exit() {
	s.timeoutID = 0
	s.spinnerTimerID = 0
}

func (s *ProcessingScreen) Name() string {
	return "Processing"
}
