//go:build screen

package screens

import "goratt/video/screen"

// OpeningScreen displays the door opening state.
type OpeningScreen struct {
	mgr      *screen.Manager
	member   string
	nickname string
	warning  string
}

// NewOpeningScreen creates a new opening screen.
func NewOpeningScreen() *OpeningScreen {
	return &OpeningScreen{}
}

// SetInfo sets the member info to display.
func (s *OpeningScreen) SetInfo(member, nickname, warning string) {
	s.member = member
	s.nickname = nickname
	s.warning = warning
}

func (s *OpeningScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
}

func (s *OpeningScreen) Update() {
	s.mgr.FillBackground(0.7, 0.7, 0) // Yellow

	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("Opening...", y, 0, 0, 0)

	// Display name
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(displayName, y+70, 0, 0, 0)
	}

	// Display warning if present
	if s.warning != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DC().SetRGB(0.7, 0, 0) // Red warning text on yellow background
		s.mgr.DC().DrawStringAnchored(s.warning, float64(s.mgr.Width()/2), y+130, 0.5, 0.5)
	}

	s.mgr.Flush()
}

func (s *OpeningScreen) HandleEvent(event screen.Event) bool {
	return false
}

func (s *OpeningScreen) Exit() {
	s.member = ""
	s.nickname = ""
	s.warning = ""
}

func (s *OpeningScreen) Name() string {
	return "Opening"
}
