//go:build screen
 
package doorlockscreens
 
import (
	"goratt/lib/video/screen"
)
 
type DoorlockIdleScreen struct {
	mgr *screen.Manager
}
 
func NewDoorlockIdleScreen() *DoorlockIdleScreen {
	return &DoorlockIdleScreen{}
}
 
func (s *DoorlockIdleScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}
 
func (s *DoorlockIdleScreen) Update() {
	s.mgr.FillBackground(0, 0, 0.2) // Deep blue
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 20
	s.mgr.DrawCentered("LOCKED", y, 1, 1, 1)
 
	s.mgr.SetFontSize(24)
	s.mgr.DrawCentered("Swipe to enter", y+70, 0.8, 0.8, 0.8)
 
	s.mgr.Flush()
}
 
func (s *DoorlockIdleScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DoorlockIdleScreen) Exit() {
}
 
func (s *DoorlockIdleScreen) Name() string {
	return "DoorlockIdle"
}
