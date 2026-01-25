//go:build screen
 
package darkroomscreens
 
import (
	"goratt/lib/video/screen"
)
 
type DarkroomOpeningScreen struct {
	mgr *screen.Manager
}
 
func NewDarkroomOpeningScreen() *DarkroomOpeningScreen {
	return &DarkroomOpeningScreen{}
}
 
func (s *DarkroomOpeningScreen) SetInfo(member, nickname, warning string) {
}
 
func (s *DarkroomOpeningScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}
 
func (s *DarkroomOpeningScreen) Update() {
	s.mgr.FillBackground(0.1, 0, 0)
	s.mgr.SetFontSize(64)
	s.mgr.DrawCentered("OPENING", float64(s.mgr.Height()/2), 0.8, 0, 0)
	s.mgr.Flush()
}
 
func (s *DarkroomOpeningScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DarkroomOpeningScreen) Exit() {
}
 
func (s *DarkroomOpeningScreen) Name() string {
	return "DarkroomOpening"
}
