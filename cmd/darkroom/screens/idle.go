//go:build screen
 
package darkroomscreens
 
import (
	"log"
	"time"
 
	"goratt/lib/video/screen"
)
 
type DarkroomIdleScreen struct {
	mgr       *screen.Manager
	summary   string
	organizer string
	when      string

	clockTimerID  screen.TimerID

	// IP address display
	lastIP        string
	ipTimerID     screen.TimerID
	ipHideTimerID screen.TimerID // Timer to auto-hide forced IP display
	ipBarHeight   int
	forceShowIP   bool // Force IP display even after startup window
	forceHideIP   bool // Force IP to hide even during startup window
	buildID       string
}

func NewDarkroomIdleScreen() *DarkroomIdleScreen {
	log.Println("DEBUG: NewDarkroomIdleScreen was called!")
	return &DarkroomIdleScreen{
		ipBarHeight: 44,
	}
}

// SetBuildID sets the build identifier string.
func (s *DarkroomIdleScreen) SetBuildID(id string) {
	s.buildID = id
}
 
func (s *DarkroomIdleScreen) SetNextReservation(summary, organizer, when string) {
	s.summary = summary
	s.organizer = organizer
	s.when = when
}
 
func (s *DarkroomIdleScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	s.startClockTimer()
}

func (s *DarkroomIdleScreen) startClockTimer() {
	// Calculate time until next minute
	now := time.Now()
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	duration := nextMinute.Sub(now)

	s.clockTimerID = s.mgr.SetTimeout(duration, func(scr screen.Screen) {
		if s.clockTimerID == 0 {
			return
		}
		s.Update()
		s.startClockTimer()
	})
}
 
func (s *DarkroomIdleScreen) Update() {
	log.Println("DEBUG: DarkroomIdleScreen.Update() is drawing to the screen!")
	s.mgr.FillBackground(0, 0.5, 0) // Green background
 
	s.mgr.SetFontSize(64)
	y := 110.0
	if s.summary == "" {
		y = float64(s.mgr.Height()/2) - 40
	}
 
	s.mgr.DrawCentered("Room Available", y, 1, 1, 1)
 
	s.mgr.SetFontSize(32)
	s.mgr.DrawCentered("Swipe fob to use room", y+50, 1, 1, 1)
 
	if s.summary != "" {
		h := y + 85
		s.mgr.FillRect(50, int(h+20), s.mgr.Width()-100, 180, 1, 1, 1)
 
		// Border/Title box for "Next Reservation"
		s.mgr.FillRect(200, int(h-5), s.mgr.Width()-400, 48, 0.3, 0.6, 0.3)
 
		s.mgr.SetFontSize(32)
		s.mgr.DC().SetRGB(0.1, 0.3, 0.1)
		s.mgr.DrawCentered("-Next Reservation-", h+17, 0.1, 0.3, 0.1)
 
		h += 55
		s.mgr.SetFontSize(32)
		s.mgr.DrawCentered(s.summary, h, 0.2, 0.5, 0.2)
		h += 40
		s.mgr.DrawCentered(s.organizer, h, 0.2, 0.5, 0.2)
		h += 40
		s.mgr.DrawCentered(s.when, h, 0.2, 0.5, 0.2)
	}
 
	// Lower Banner with time
	timeYPos := float64(s.mgr.Height() - 76)
	s.mgr.FillRect(0, int(timeYPos), s.mgr.Width(), 66, 0.3, 0.5, 0.3)
	s.mgr.SetFontSize(42)
	formattedDateTime := time.Now().Format("Jan 2 3:04pm")
	s.mgr.DrawCentered(formattedDateTime, timeYPos+33, 0, 0.25, 0)
 
	s.mgr.Flush()
}
 
func (s *DarkroomIdleScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *DarkroomIdleScreen) Exit() {
	s.clockTimerID = 0
}
 
func (s *DarkroomIdleScreen) Name() string {
	return "DarkroomIdle"
}
