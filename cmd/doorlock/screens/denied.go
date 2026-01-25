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
	s.mgr.FillBackground(0.5, 0, 0) // Red
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 20
	s.mgr.DrawCentered("DENIED", y, 1, 1, 1)
 
	if s.warning != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DrawCentered(s.warning, y+70, 0.9, 0.9, 0.9)
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
