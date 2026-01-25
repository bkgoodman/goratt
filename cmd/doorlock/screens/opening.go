//go:build screen
 
package doorlockscreens
 
import (
	"goratt/lib/video/screen"
)
 
type DoorlockOpeningScreen struct {
	mgr *screen.Manager
}
 
func NewDoorlockOpeningScreen() *DoorlockOpeningScreen {
	return &DoorlockOpeningScreen{}
}
 
func (s *DoorlockOpeningScreen) SetInfo(member, nickname, warning string) {
}
 
func (s *DoorlockOpeningScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}
 
func (s *DoorlockOpeningScreen) Update() {
	s.mgr.FillBackground(0, 0, 0.4) // Blue
	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("OPENING", float64(s.mgr.Height()/2), 1, 1, 1)
	s.mgr.Flush()
}
 
func (s *DoorlockOpeningScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DoorlockOpeningScreen) Exit() {
}
 
func (s *DoorlockOpeningScreen) Name() string {
	return "DoorlockOpening"
}
