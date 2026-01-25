//go:build !screen
 
package doorlockscreens
 
import "goratt/lib/video/screen"
 
type DoorlockIdleScreen struct{}
func NewDoorlockIdleScreen() *DoorlockIdleScreen { return &DoorlockIdleScreen{} }
func (s *DoorlockIdleScreen) Init(mgr *screen.Manager) {}
func (s *DoorlockIdleScreen) Update() {}
func (s *DoorlockIdleScreen) HandleEvent(event screen.Event) bool { return false }
func (s *DoorlockIdleScreen) Exit() {}
func (s *DoorlockIdleScreen) Name() string { return "Idle" }
 
type DoorlockGrantedScreen struct{}
func NewDoorlockGrantedScreen() *DoorlockGrantedScreen { return &DoorlockGrantedScreen{} }
func (s *DoorlockGrantedScreen) SetInfo(member, nickname, warning string) {}
func (s *DoorlockGrantedScreen) Init(mgr *screen.Manager) {}
func (s *DoorlockGrantedScreen) Update() {}
func (s *DoorlockGrantedScreen) HandleEvent(event screen.Event) bool { return false }
func (s *DoorlockGrantedScreen) Exit() {}
func (s *DoorlockGrantedScreen) Name() string { return "Granted" }
 
type DoorlockDeniedScreen struct{}
func NewDoorlockDeniedScreen() *DoorlockDeniedScreen { return &DoorlockDeniedScreen{} }
func (s *DoorlockDeniedScreen) SetInfo(member, nickname, warning string) {}
func (s *DoorlockDeniedScreen) Init(mgr *screen.Manager) {}
func (s *DoorlockDeniedScreen) Update() {}
func (s *DoorlockDeniedScreen) HandleEvent(event screen.Event) bool { return false }
func (s *DoorlockDeniedScreen) Exit() {}
func (s *DoorlockDeniedScreen) Name() string { return "Denied" }
 
type DoorlockOpeningScreen struct{}
func NewDoorlockOpeningScreen() *DoorlockOpeningScreen { return &DoorlockOpeningScreen{} }
func (s *DoorlockOpeningScreen) SetInfo(member, nickname, warning string) {}
func (s *DoorlockOpeningScreen) Init(mgr *screen.Manager) {}
func (s *DoorlockOpeningScreen) Update() {}
func (s *DoorlockOpeningScreen) HandleEvent(event screen.Event) bool { return false }
func (s *DoorlockOpeningScreen) Exit() {}
func (s *DoorlockOpeningScreen) Name() string { return "Opening" }
