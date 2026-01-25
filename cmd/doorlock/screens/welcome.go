//go:build screen
 
package doorlockscreens
 
import (
	"time"
 
	"goratt/lib/video/screen"
)
 
type DoorlockGrantedScreen struct {
	mgr      *screen.Manager
	member   string
	nickname string
}
 
func NewDoorlockGrantedScreen() *DoorlockGrantedScreen {
	return &DoorlockGrantedScreen{}
}
 
func (s *DoorlockGrantedScreen) SetInfo(member, nickname, warning string) {
	s.member = member
	s.nickname = nickname
}
 
func (s *DoorlockGrantedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	mgr.SetTimeout(5*time.Second, func(scr screen.Screen) {
		mgr.SwitchTo(screen.ScreenIdle)
	})
}
 
func (s *DoorlockGrantedScreen) Update() {
	s.mgr.FillBackground(0, 0.4, 0) // Green but for doorlock
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("WELCOME", y, 1, 1, 1)
 
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(displayName, y+70, 1, 1, 1)
	}
 
	s.mgr.Flush()
}
 
func (s *DoorlockGrantedScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DoorlockGrantedScreen) Exit() {
}
 
func (s *DoorlockGrantedScreen) Name() string {
	return "DoorlockGranted"
}
