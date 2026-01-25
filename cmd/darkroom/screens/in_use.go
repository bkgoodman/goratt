//go:build screen

package darkroomscreens

import (
	"goratt/lib/video/screen"
)

type RoomInUseScreen struct {
	mgr       *screen.Manager
	member    string
	nickname  string
	timeoutID screen.TimerID
}

func NewRoomInUseScreen() *RoomInUseScreen {
	return &RoomInUseScreen{}
}

func (s *RoomInUseScreen) SetInfo(member, nickname string) {
	s.member = member
	s.nickname = nickname
}

func (s *RoomInUseScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	// Show for 30 seconds then back to idle (or however long a darkroom stay is?)
	// Usually darkroom stay is long - maybe it should stay until badge-out?
	// But let's start with a timeout for now.
}

func (s *RoomInUseScreen) Update() {
	s.mgr.FillBackground(0.5, 0, 0) // Deep red (safe for darkroom!)

	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("ROOM IN USE", y, 1, 1, 1)

	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(displayName, y+80, 1, 1, 1)
	}

	s.mgr.SetFontSize(24)
	s.mgr.DrawCentered("Please be careful", y+140, 0.8, 0.8, 0.8)

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
}

func (s *RoomInUseScreen) Name() string {
	return "RoomInUse"
}
