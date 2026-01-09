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

// RFIDData contains data for RFID-related events.
type RFIDData struct {
	TagID    uint64
	Member   string
	Nickname string
	Warning  string
	Allowed  bool
	Found    bool
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

func (e Event) RFID() *RFIDData {
	if data, ok := e.Data.(RFIDData); ok {
		return &data
	}
	return nil
}

func (e Event) Rotary() *RotaryData {
	if data, ok := e.Data.(RotaryData); ok {
		return &data
	}
	return nil
}

func (e Event) Pin() *PinData {
	if data, ok := e.Data.(PinData); ok {
		return &data
	}
	return nil
}

// MQTTData contains data for MQTT events.
type MQTTData struct {
	Topic   string
	Payload []byte
}

func (e Event) MQTT() *MQTTData {
	if data, ok := e.Data.(MQTTData); ok {
		return &data
	}
	return nil
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
	ScreenSelectAmount
	ScreenConfirm
	ScreenAborted
)

// TimerID uniquely identifies a timer.
type TimerID uint64

// TimerCallback is called when a timer fires.
type TimerCallback func(screen Screen)

// Manager is a stub when screen support is not compiled in.
type Manager struct{}

func NewManager() *Manager                                                   { return nil }
func (m *Manager) Register(id ScreenID, s Screen)                            {}
func (m *Manager) SwitchTo(id ScreenID)                                      {}
func (m *Manager) Current() Screen                                           { return nil }
func (m *Manager) SendEvent(event Event) bool                                { return false }
func (m *Manager) Update()                                                   {}
func (m *Manager) Flush()                                                    {}
func (m *Manager) SetTimeout(d time.Duration, cb TimerCallback) TimerID      { return 0 }
func (m *Manager) ClearTimeout(id TimerID) bool                              { return false }
func (m *Manager) ClearAllTimeouts()                                         {}
func (m *Manager) SetMQTTConnected(connected bool)                           {}
func (m *Manager) IsMQTTConnected() bool                                     { return false }
func (m *Manager) SetVendingSession(member, nickname string, amount float64) {}
func (m *Manager) GetVendingSession() (string, string, float64)              { return "", "", 0 }
func (m *Manager) ClearVendingSession()                                      {}
