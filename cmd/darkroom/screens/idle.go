//go:build screen
 
package darkroomscreens
 
import (
	"goratt/lib/video/screen"
)
 
type DarkroomIdleScreen struct {
	mgr *screen.Manager
}
 
func NewDarkroomIdleScreen() *DarkroomIdleScreen {
	return &DarkroomIdleScreen{}
}
 
func (s *DarkroomIdleScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}
 
func (s *DarkroomIdleScreen) Update() {
	s.mgr.FillBackground(0.1, 0, 0) // Dim red for safety
 
	s.mgr.SetFontSize(56)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("DARKROOM", y, 0.8, 0, 0)
 
	s.mgr.SetFontSize(24)
	s.mgr.DrawCentered("Swipe to enter", y+70, 0.6, 0, 0)
 
	s.mgr.Flush()
}
 
func (s *DarkroomIdleScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DarkroomIdleScreen) Exit() {
}
 
func (s *DarkroomIdleScreen) Name() string {
	return "DarkroomIdle"
}
