//go:build screen

package screens

import (
	"goratt/video/screen"
)

// GrantedScreen displays the access granted state.
type GrantedScreen struct {
	mgr      *screen.Manager
	member   string
	nickname string
	warning  string
}

// NewGrantedScreen creates a new granted screen.
func NewGrantedScreen() *GrantedScreen {
	return &GrantedScreen{}
}

// SetInfo sets the member info to display.
func (s *GrantedScreen) SetInfo(member, nickname, warning string) {
	s.member = member
	s.nickname = nickname
	s.warning = warning
}

func (s *GrantedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}

func (s *GrantedScreen) Update() {
	s.mgr.FillBackground(0, 0.7, 0) // Bright green

	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("Access Granted", y, 1, 1, 1)

	// Display name (prefer nickname, fall back to member)
	s.mgr.SetFontSize(48)
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.DrawCentered(displayName, y+70, 1, 1, 1)
	}

	// Display warning if present
	if s.warning != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DC().SetRGB(1, 1, 0) // Yellow warning text
		s.mgr.DC().DrawStringAnchored(s.warning, float64(s.mgr.Width()/2), y+130, 0.5, 0.5)
	}

	s.mgr.Flush()
}

func (s *GrantedScreen) HandleEvent(event screen.Event) bool {
	// Could handle button press to dismiss early
	return false
}

func (s *GrantedScreen) Exit() {
	s.member = ""
	s.nickname = ""
	s.warning = ""
}

func (s *GrantedScreen) Name() string {
	return "Granted"
}
