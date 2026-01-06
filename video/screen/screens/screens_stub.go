//go:build !screen

package screens

import "goratt/video/screen"

// IdleScreen stub
type IdleScreen struct{}

func NewIdleScreen() *IdleScreen                          { return &IdleScreen{} }
func (s *IdleScreen) Init(mgr *screen.Manager)            {}
func (s *IdleScreen) Update()                             {}
func (s *IdleScreen) HandleEvent(event screen.Event) bool { return false }
func (s *IdleScreen) Exit()                               {}
func (s *IdleScreen) Name() string                        { return "Idle" }

// GrantedScreen stub
type GrantedScreen struct{}

func NewGrantedScreen() *GrantedScreen                            { return &GrantedScreen{} }
func (s *GrantedScreen) SetInfo(member, nickname, warning string) {}
func (s *GrantedScreen) Init(mgr *screen.Manager)                 {}
func (s *GrantedScreen) Update()                                  {}
func (s *GrantedScreen) HandleEvent(event screen.Event) bool      { return false }
func (s *GrantedScreen) Exit()                                    {}
func (s *GrantedScreen) Name() string                             { return "Granted" }

// DeniedScreen stub
type DeniedScreen struct{}

func NewDeniedScreen() *DeniedScreen                             { return &DeniedScreen{} }
func (s *DeniedScreen) SetInfo(member, nickname, warning string) {}
func (s *DeniedScreen) Init(mgr *screen.Manager)                 {}
func (s *DeniedScreen) Update()                                  {}
func (s *DeniedScreen) HandleEvent(event screen.Event) bool      { return false }
func (s *DeniedScreen) Exit()                                    {}
func (s *DeniedScreen) Name() string                             { return "Denied" }

// OpeningScreen stub
type OpeningScreen struct{}

func NewOpeningScreen() *OpeningScreen                            { return &OpeningScreen{} }
func (s *OpeningScreen) SetInfo(member, nickname, warning string) {}
func (s *OpeningScreen) Init(mgr *screen.Manager)                 {}
func (s *OpeningScreen) Update()                                  {}
func (s *OpeningScreen) HandleEvent(event screen.Event) bool      { return false }
func (s *OpeningScreen) Exit()                                    {}
func (s *OpeningScreen) Name() string                             { return "Opening" }

// ConnectionLostScreen stub
type ConnectionLostScreen struct{}

func NewConnectionLostScreen() *ConnectionLostScreen                { return &ConnectionLostScreen{} }
func (s *ConnectionLostScreen) Init(mgr *screen.Manager)            {}
func (s *ConnectionLostScreen) Update()                             {}
func (s *ConnectionLostScreen) HandleEvent(event screen.Event) bool { return false }
func (s *ConnectionLostScreen) Exit()                               {}
func (s *ConnectionLostScreen) Name() string                        { return "ConnectionLost" }

// ShutdownScreen stub
type ShutdownScreen struct{}

func NewShutdownScreen() *ShutdownScreen                      { return &ShutdownScreen{} }
func (s *ShutdownScreen) Init(mgr *screen.Manager)            {}
func (s *ShutdownScreen) Update()                             {}
func (s *ShutdownScreen) HandleEvent(event screen.Event) bool { return false }
func (s *ShutdownScreen) Exit()                               {}
func (s *ShutdownScreen) Name() string                        { return "Shutdown" }
