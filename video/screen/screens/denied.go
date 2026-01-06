//go:build screen

package screens

import (
	"time"

	"goratt/video/screen"
)

// DeniedScreen displays the access denied state.
type DeniedScreen struct {
	mgr       *screen.Manager
	member    string
	nickname  string
	warning   string
	timeout   time.Duration
	onDismiss func() // Called when screen is dismissed (timeout or button)
}

// NewDeniedScreen creates a new denied screen.
func NewDeniedScreen() *DeniedScreen {
	return &DeniedScreen{
		timeout: 3 * time.Second, // Default timeout
	}
}

// SetInfo sets the member info to display.
func (s *DeniedScreen) SetInfo(member, nickname, warning string) {
	s.member = member
	s.nickname = nickname
	s.warning = warning
}

// SetTimeout sets how long to display before auto-dismissing.
func (s *DeniedScreen) SetTimeout(d time.Duration) {
	s.timeout = d
}

// SetOnDismiss sets a callback to be called when the screen is dismissed.
func (s *DeniedScreen) SetOnDismiss(fn func()) {
	s.onDismiss = fn
}

func (s *DeniedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr

	// Set timeout to auto-dismiss to idle
	mgr.SetTimeout(s.timeout, func(scr screen.Screen) {
		s.dismiss()
	})
}

func (s *DeniedScreen) dismiss() {
	if s.onDismiss != nil {
		s.onDismiss()
	}
	s.mgr.SwitchTo(screen.ScreenIdle)
}

func (s *DeniedScreen) Update() {
	s.mgr.FillBackground(0.7, 0, 0) // Red

	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("Access Denied", y, 1, 1, 1)

	// Display name if known
	displayName := s.nickname
	if displayName == "" {
		displayName = s.member
	}
	if displayName != "" {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(displayName, y+70, 1, 1, 1)
	}

	// Display warning/reason if present
	if s.warning != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DC().SetRGB(1, 1, 0) // Yellow warning text
		s.mgr.DC().DrawStringAnchored(s.warning, float64(s.mgr.Width()/2), y+130, 0.5, 0.5)
	}

	s.mgr.Flush()
}

func (s *DeniedScreen) HandleEvent(event screen.Event) bool {
	// Rotary button press dismisses early
	if event.Type == screen.EventRotaryPress {
		s.dismiss()
		return true
	}
	return false
}

func (s *DeniedScreen) Exit() {
	s.member = ""
	s.nickname = ""
	s.warning = ""
	s.onDismiss = nil
}

func (s *DeniedScreen) Name() string {
	return "Denied"
}
