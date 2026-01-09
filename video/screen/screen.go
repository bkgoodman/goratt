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
	EventRotaryLongPress  // Rotary button held for >1s
	EventPin              // GPIO pin event
	EventMQTTConnected    // MQTT broker connected/reconnected
	EventMQTTDisconnected // MQTT broker disconnected
	EventMQTTMessage      // MQTT message received
)

// RotaryID identifies a specific rotary encoder
type RotaryID int

const (
	RotaryMain RotaryID = iota // Main/default rotary encoder
	RotaryAux                  // Auxiliary rotary encoder
)

// PinID identifies a specific GPIO pin input
type PinID int

const (
	PinButton1   PinID = iota // Primary button
	PinButton2                // Secondary button
	PinSensor1                // Sensor input 1
	PinSensor2                // Sensor input 2
	PinEstop                  // Emergency stop
	PinDoor                   // Door sensor
	PinSafelight              // Safelight switch
	PinActivity               // Activity switch or Current Sense
	PinEnable                 // Enable switch (On/Off)
)

// Event is the base event structure. Type-specific data is in the Data field.
type Event struct {
	Type EventType
	Data any // Type-specific event data (RFIDData, RotaryData, PinData, etc.)
}

// RFIDData contains data for RFID-related events (EventRFID, EventAuthorized, EventDenied).
type RFIDData struct {
	TagID    uint64
	Member   string
	Nickname string
	Warning  string
	Allowed  bool // ACL allowed flag
	Found    bool // Whether tag was found in ACL
}

// RotaryData contains data for rotary encoder events.
type RotaryData struct {
	ID    RotaryID // Which rotary encoder
	Delta int      // +1 for CW, -1 for CCW (for turn events)
}

// PinData contains data for GPIO pin events.
type PinData struct {
	ID      PinID // Which pin
	Pressed bool  // true for press/active, false for release/inactive
}

// Helper methods to extract typed data from events

// RFID returns the RFIDData from the event, or nil if not an RFID event.
func (e Event) RFID() *RFIDData {
	if data, ok := e.Data.(RFIDData); ok {
		return &data
	}
	return nil
}

// Rotary returns the RotaryData from the event, or nil if not a rotary event.
func (e Event) Rotary() *RotaryData {
	if data, ok := e.Data.(RotaryData); ok {
		return &data
	}
	return nil
}

// Pin returns the PinData from the event, or nil if not a pin event.
func (e Event) Pin() *PinData {
	if data, ok := e.Data.(PinData); ok {
		return &data
	}
	return nil
}

// MQTTData contains data for MQTT events.
type MQTTData struct {
	Topic   string // Topic for message events
	Payload []byte // Payload for message events
}

// MQTT returns the MQTTData from the event, or nil if not an MQTT message event.
func (e Event) MQTT() *MQTTData {
	if data, ok := e.Data.(MQTTData); ok {
		return &data
	}
	return nil
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
	ScreenSelectAmount
	ScreenConfirm
	ScreenAborted
)
