//go:build screen
 
package vendingscreens
 
import (
	"time"
 
	"goratt/lib/video/screen"
	"goratt/cmd/vending/assets"
)
 
type VendingDeniedScreen struct {
	mgr     *screen.Manager
	warning string
}
 
func NewVendingDeniedScreen() *VendingDeniedScreen {
	return &VendingDeniedScreen{}
}
 
func (s *VendingDeniedScreen) SetInfo(member, nickname, warning string) {
	s.warning = warning
}
 
func (s *VendingDeniedScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	mgr.SetTimeout(3*time.Second, func(scr screen.Screen) {
		mgr.SwitchTo(screen.ScreenIdle)
	})

	// Play purchase audio
	s.mgr.PlayAudioBytes(assets.Audio_notrecognized)
}
 
func (s *VendingDeniedScreen) Update() {
	s.mgr.FillBackground(0.5, 0, 0)
 
	s.mgr.SetFontSize(64)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("NOT ALLOWED", y, 1, 1, 1)
 
	if s.warning != "" {
		s.mgr.SetFontSize(32)
		s.mgr.DrawCentered(s.warning, y+70, 0.9, 0.9, 0.9)
	}
 
	s.mgr.Flush()
}
 
func (s *VendingDeniedScreen) HandleEvent(event screen.Event) bool {
	return false
}
 
func (s *VendingDeniedScreen) Exit() {
}
 
func (s *VendingDeniedScreen) Name() string {
	return "VendingDenied"
}
