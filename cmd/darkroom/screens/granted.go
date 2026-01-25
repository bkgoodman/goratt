//go:build screen
 
package darkroomscreens
 
import (
	"time"
 
	"goratt/lib/video/screen"
)
 
type DarkroomGrantedScreen struct {
	mgr      *screen.Manager
	member   string
	nickname string
	warning  string
}
 
func NewDarkroomGrantedScreen() *DarkroomGrantedScreen {
	return &DarkroomGrantedScreen{}
}
 
func (s *DarkroomGrantedScreen) SetInfo(member, nickname, warning string) {
	s.member = member
	s.nickname = nickname
	s.warning = warning
}
 
func (s *DarkroomGrantedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	mgr.SetTimeout(5*time.Second, func(scr screen.Screen) {
		// Manager will usually be switched to RoomInUse by the app handler
	})
}
 
func (s *DarkroomGrantedScreen) Update() {
	s.mgr.FillBackground(0.4, 0, 0) // Solid red
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("Welcome", y, 1, 0, 0)
 
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(displayName, y+70, 1, 0, 0)
	}
 
	s.mgr.Flush()
}
 
func (s *DarkroomGrantedScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DarkroomGrantedScreen) Exit() {
}
 
func (s *DarkroomGrantedScreen) Name() string {
	return "DarkroomGranted"
}
