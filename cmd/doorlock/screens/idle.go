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
	s.mgr.FillBackground(0, 0.5, 0) // Green background
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height() / 2)
	s.mgr.DrawCentered("Ready", y, 1, 1, 1)
 
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
