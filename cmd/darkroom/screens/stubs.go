//go:build !screen
 
package darkroomscreens
 
import "goratt/lib/video/screen"
 
type DarkroomIdleScreen struct{}
func NewDarkroomIdleScreen() *DarkroomIdleScreen { return &DarkroomIdleScreen{} }
func (s *DarkroomIdleScreen) Init(mgr *screen.Manager) {}
func (s *DarkroomIdleScreen) Update() {}
func (s *DarkroomIdleScreen) HandleEvent(event screen.Event) bool { return false }
func (s *DarkroomIdleScreen) Exit() {}
func (s *DarkroomIdleScreen) Name() string { return "Idle" }
 
type DarkroomGrantedScreen struct{}
func NewDarkroomGrantedScreen() *DarkroomGrantedScreen { return &DarkroomGrantedScreen{} }
func (s *DarkroomGrantedScreen) SetInfo(member, nickname, warning string) {}
func (s *DarkroomGrantedScreen) Init(mgr *screen.Manager) {}
func (s *DarkroomGrantedScreen) Update() {}
func (s *DarkroomGrantedScreen) HandleEvent(event screen.Event) bool { return false }
func (s *DarkroomGrantedScreen) Exit() {}
func (s *DarkroomGrantedScreen) Name() string { return "Granted" }
 
type DarkroomDeniedScreen struct{}
func NewDarkroomDeniedScreen() *DarkroomDeniedScreen { return &DarkroomDeniedScreen{} }
func (s *DarkroomDeniedScreen) SetInfo(member, nickname, warning string) {}
func (s *DarkroomDeniedScreen) Init(mgr *screen.Manager) {}
func (s *DarkroomDeniedScreen) Update() {}
func (s *DarkroomDeniedScreen) HandleEvent(event screen.Event) bool { return false }
func (s *DarkroomDeniedScreen) Exit() {}
func (s *DarkroomDeniedScreen) Name() string { return "Denied" }
 
type DarkroomOpeningScreen struct{}
func NewDarkroomOpeningScreen() *DarkroomOpeningScreen { return &DarkroomOpeningScreen{} }
func (s *DarkroomOpeningScreen) SetInfo(member, nickname, warning string) {}
func (s *DarkroomOpeningScreen) Init(mgr *screen.Manager) {}
func (s *DarkroomOpeningScreen) Update() {}
func (s *DarkroomOpeningScreen) HandleEvent(event screen.Event) bool { return false }
func (s *DarkroomOpeningScreen) Exit() {}
func (s *DarkroomOpeningScreen) Name() string { return "Opening" }
 
type RoomInUseScreen struct{}
func NewRoomInUseScreen() *RoomInUseScreen { return &RoomInUseScreen{} }
func (s *RoomInUseScreen) SetInfo(member, nickname string) {}
func (s *RoomInUseScreen) Init(mgr *screen.Manager) {}
func (s *RoomInUseScreen) Update() {}
func (s *RoomInUseScreen) HandleEvent(event screen.Event) bool { return false }
func (s *RoomInUseScreen) Exit() {}
func (s *RoomInUseScreen) Name() string { return "RoomInUse" }
 
type SafeLightScreen struct{}
func NewSafeLightScreen() *SafeLightScreen { return &SafeLightScreen{} }
func (s *SafeLightScreen) Init(mgr *screen.Manager) {}
func (s *SafeLightScreen) Update() {}
func (s *SafeLightScreen) HandleEvent(event screen.Event) bool { return false }
func (s *SafeLightScreen) Exit() {}
func (s *SafeLightScreen) Name() string { return "SafeLight" }
