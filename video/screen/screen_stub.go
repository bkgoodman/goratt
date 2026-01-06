//go:build !screen

package screen

import "time"

// Event types that screens can receive
type EventType int

const (
	EventRFID       EventType = iota // Raw RFID swipe (before ACL lookup)
	EventAuthorized                  // ACL lookup succeeded and user is allowed
	EventDenied                      // ACL lookup failed or user not allowed
	EventRotaryTurn
	EventRotaryPress
	EventButton
)

// Event represents an input event sent to a screen.
type Event struct {
	Type     EventType
	TagID    uint64
	Member   string
	Nickname string
	Warning  string
	Allowed  bool
	Found    bool
	Delta    int
}

// Screen is the interface that all screens must implement.
type Screen interface {
	Init(mgr *Manager)
	Update()
	HandleEvent(event Event) bool
	Exit()
	Name() string
}

// ScreenID identifies a screen type.
type ScreenID int

const (
	ScreenIdle ScreenID = iota
	ScreenGranted
	ScreenDenied
	ScreenOpening
	ScreenConnectionLost
	ScreenShutdown
)

// TimerID uniquely identifies a timer.
type TimerID uint64

// TimerCallback is called when a timer fires.
type TimerCallback func(screen Screen)

// Manager is a stub when screen support is not compiled in.
type Manager struct{}

func NewManager() *Manager                                              { return nil }
func (m *Manager) Register(id ScreenID, s Screen)                       {}
func (m *Manager) SwitchTo(id ScreenID)                                 {}
func (m *Manager) Current() Screen                                      { return nil }
func (m *Manager) SendEvent(event Event) bool                           { return false }
func (m *Manager) Update()                                              {}
func (m *Manager) Flush()                                               {}
func (m *Manager) SetTimeout(d time.Duration, cb TimerCallback) TimerID { return 0 }
func (m *Manager) ClearTimeout(id TimerID) bool                         { return false }
func (m *Manager) ClearAllTimeouts()                                    {}
