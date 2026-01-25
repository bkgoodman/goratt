//go:build screen
 
package darkroomscreens
 
import (
	"time"
 
	"goratt/lib/video/screen"
)
 
type DarkroomDeniedScreen struct {
	mgr      *screen.Manager
	member   string
	nickname string
	warning  string
}
 
func NewDarkroomDeniedScreen() *DarkroomDeniedScreen {
	return &DarkroomDeniedScreen{}
}
 
func (s *DarkroomDeniedScreen) SetInfo(member, nickname, warning string) {
	s.member = member
	s.nickname = nickname
	s.warning = warning
}
 
func (s *DarkroomDeniedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	mgr.SetTimeout(3*time.Second, func(scr screen.Screen) {
		mgr.SwitchTo(screen.ScreenIdle)
	})
}
 
func (s *DarkroomDeniedScreen) Update() {
	s.mgr.FillBackground(1, 0, 0) // Pulsing/Bright red
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("DENIED", y, 0, 0, 0)
 
	if s.warning != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DrawCentered(s.warning, y+70, 0, 0, 0)
	}
 
	s.mgr.Flush()
}
 
func (s *DarkroomDeniedScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DarkroomDeniedScreen) Exit() {
}
 
func (s *DarkroomDeniedScreen) Name() string {
	return "DarkroomDenied"
}
