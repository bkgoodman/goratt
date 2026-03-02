//go:build screen
 
package darkroomscreens
 
import (
	"fmt"
	"strings"
	"time"
 
	"goratt/lib/video/screen"
)
 
type RoomInUseScreen struct {
	mgr       *screen.Manager
	member    string
	startTime time.Time
	timerID   screen.TimerID
}
 
func NewRoomInUseScreen() *RoomInUseScreen {
	return &RoomInUseScreen{}
}
 
func (s *RoomInUseScreen) SetInfo(member, nickname string) {
	s.member = member
	s.startTime = time.Now()
}
 
func (s *RoomInUseScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	s.scheduleUpdate()
}
 
func (s *RoomInUseScreen) scheduleUpdate() {
	s.timerID = s.mgr.SetTimeout(time.Minute, func(scr screen.Screen) {
		s.Update()
		s.scheduleUpdate()
	})
}
 
func (s *RoomInUseScreen) howLongAgo() string {
	m := int(time.Since(s.startTime).Minutes())
	if m == 1 {
		return "In-Use for 1 minute"
	}
	return fmt.Sprintf("In-Use for %d minutes", m)
}
 
func (s *RoomInUseScreen) Update() {
	s.mgr.FillBackground(0, 0, 0.5) // blue background (original branch)
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("Room in Use", y, 1, 1, 1)
 
	s.mgr.SetFontSize(28)
	s.mgr.DrawCentered("Swipe fob to badge-out of room", y+50, 1, 1, 1)
 
	// Duration banner at bottom
	timeYPos := float64(s.mgr.Height() - 76)
	s.mgr.FillRect(0, int(timeYPos), s.mgr.Width(), 66, 0.2, 0.2, 0.5)
	s.mgr.SetFontSize(42)
	s.mgr.DrawCentered(s.howLongAgo(), timeYPos+33, 0, 0.8, 0)
 
	displayName := strings.ReplaceAll(s.member, ".", " ")
	if displayName != "" {
		// name banner at top
		s.mgr.FillRect(0, 10, s.mgr.Width(), 66, 0.2, 0.2, 0.5)
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(displayName, 43, 0.5, 0.5, 0.5)
	}
 
	s.mgr.Flush()
}
 
func (s *RoomInUseScreen) HandleEvent(event screen.Event) bool {
	if event.Type == screen.EventRotaryPress {
		s.mgr.SwitchTo(screen.ScreenIdle)
		return true
	}
	return false
}
 
func (s *RoomInUseScreen) Exit() {
	s.timerID = 0
}
 
func (s *RoomInUseScreen) Name() string {
	return "RoomInUse"
}
