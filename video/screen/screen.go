//go:build screen

package screen

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
	Type EventType

	// For RFID/Authorization events
	TagID    uint64
	Member   string
	Nickname string
	Warning  string
	Allowed  bool // For EventRFID: ACL allowed flag; for EventAuthorized/Denied: always true/false
	Found    bool // Whether tag was found in ACL

	// For rotary events
	Delta int // +1 for CW, -1 for CCW
}

// Screen is the interface that all screens must implement.
type Screen interface {
	// Init is called when entering this screen.
	// The manager is provided so screens can switch to other screens.
	Init(mgr *Manager)

	// Update redraws the screen. Called after Init and whenever
	// the screen needs to refresh its display.
	Update()

	// HandleEvent processes an input event.
	// Returns true if the event was handled.
	HandleEvent(event Event) bool

	// Exit is called when leaving this screen.
	// Use for cleanup of screen-specific resources.
	Exit()

	// Name returns the screen name for debugging/logging.
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
