//go:build screen
 
package doorlockscreens
 
import (
	"time"
 
	"goratt/lib/video/screen"
)
 
type DoorlockDeniedScreen struct {
	mgr     *screen.Manager
	warning string
}
 
func NewDoorlockDeniedScreen() *DoorlockDeniedScreen {
	return &DoorlockDeniedScreen{}
}
 
func (s *DoorlockDeniedScreen) SetInfo(member, nickname, warning string) {
	s.warning = warning
}
 
func (s *DoorlockDeniedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	mgr.SetTimeout(3*time.Second, func(scr screen.Screen) {
		mgr.SwitchTo(screen.ScreenIdle)
	})
}
 
func (s *DoorlockDeniedScreen) Update() {
	s.mgr.FillBackground(0.7, 0, 0) // Red
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("Access Denied", y, 1, 1, 1)
 
	if s.warning != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DC().SetRGB(1, 1, 0) // Yellow warning text
		s.mgr.DC().DrawStringAnchored(s.warning, float64(s.mgr.Width()/2), y+130, 0.5, 0.5)
	}
 
	s.mgr.Flush()
}
 
func (s *DoorlockDeniedScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DoorlockDeniedScreen) Exit() {
}
 
func (s *DoorlockDeniedScreen) Name() string {
	return "DoorlockDenied"
}
