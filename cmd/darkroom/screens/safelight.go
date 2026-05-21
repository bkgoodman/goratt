//go:build screen
 
package darkroomscreens
 
import (
	"time"

	"goratt/lib/video/screen"
)
 
type SafeLightScreen struct {
	mgr        *screen.Manager
	timerID    screen.TimerID
	blinkState bool
}
 
func NewSafeLightScreen() *SafeLightScreen {
	return &SafeLightScreen{}
}
 
func (s *SafeLightScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	s.blinkState = false
	s.startTimer()
}

func (s *SafeLightScreen) startTimer() {
	s.timerID = s.mgr.SetTimeout(500*time.Millisecond, func(scr screen.Screen) {
		if s.timerID == 0 {
			return
		}
		s.blinkState = !s.blinkState
		s.Update()
		s.startTimer()
	})
}
 
func (s *SafeLightScreen) Update() {
	bgR, bgG, bgB := 1.0, 1.0, 0.0 // Yellow background
	fgR, fgG, fgB := 1.0, 0.0, 0.0 // Red text

	if s.blinkState {
		// Invert colors to blink (Red background, Yellow text)
		bgR, bgG, bgB = 1.0, 0.0, 0.0
		fgR, fgG, fgB = 1.0, 1.0, 0.0
	}

	s.mgr.FillBackground(bgR, bgG, bgB)
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("SAFE-LIGHT ON", y, fgR, fgG, fgB)
 
	s.mgr.SetFontSize(48)
	s.mgr.DrawCentered("DO NOT ENTER", y+80, fgR, fgG, fgB)
 
	s.mgr.Flush()
}
 
func (s *SafeLightScreen) HandleEvent(event screen.Event) bool {
	if event.Type == screen.EventRotaryPress {
		s.mgr.SwitchTo(screen.ScreenIdle)
		return true
	}
	return false
}
 
func (s *SafeLightScreen) Exit() {
	s.timerID = 0
}
 
func (s *SafeLightScreen) Name() string {
	return "SafeLight"
}
