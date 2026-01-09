//go:build screen

package screen

import (
	"log"
	"sync"
	"time"

	"github.com/fogleman/gg"
)

// TimerID uniquely identifies a timer.
type TimerID uint64

// TimerCallback is called when a timer fires.
// The screen parameter is the screen that was current when the timer was set.
type TimerCallback func(screen Screen)

// screenTimer holds timer state.
type screenTimer struct {
	id       TimerID
	timer    *time.Timer
	screen   Screen
	callback TimerCallback
}

// Manager manages screen state and transitions.
//
// # Mutex (mu) Usage
//
// The mutex protects: current, screens, timers, nextTimerID, mqttConnected.
//
// IMPORTANT: Screen callbacks (Init, Update, Exit, HandleEvent) may be called
// in two contexts:
//   - With mutex HELD: Exit() is called from SwitchTo() while holding the lock
//   - With mutex RELEASED: Init(), Update(), HandleEvent() are called after releasing
//
// Rules for Screen implementations:
//   - NEVER call ClearTimeout() from Exit() - causes deadlock (mutex already held)
//   - The manager clears all timers for a screen BEFORE calling Exit()
//   - To stop a recurring timer, just set your timerID field to 0 in Exit()
//   - Timer callbacks check if timer still exists before running user callback
//
// Rules for timer callbacks:
//   - Callbacks run with mutex RELEASED (safe to call SetTimeout, etc.)
//   - Check your screen's timerID field to detect if Exit() was called
//   - Do NOT call Current() to check if screen is active (acquires mutex,
//     potential deadlock if SwitchTo is waiting)
//
// Thread safety:
//   - SetTimeout, ClearTimeout, SwitchTo, SendEvent are safe to call from any goroutine
//   - Drawing methods (DC, Flush, FlushRect, etc.) are NOT mutex-protected;
//     only the current screen should draw, and only from its callbacks
type Manager struct {
	mu            sync.Mutex // Protects current, screens, timers, nextTimerID, mqttConnected
	current       Screen
	screens       map[ScreenID]Screen
	dc            *gg.Context
	width, height int
	updateFn      func()               // Called after drawing to flush full framebuffer
	updateRectFn  func(x, y, w, h int) // Called to flush a rectangle only

	// Timer management
	nextTimerID TimerID
	timers      map[TimerID]*screenTimer

	// App-level state that persists across screen switches
	mqttConnected bool

	// Vending session state
	vendingMember    string
	vendingNickname  string
	vendingAmount    float64 // Selected purchase amount in dollars
	vendingBalance   float64 // Current account balance
	vendingAddAmount float64 // Amount to add to account
}

// NewManager creates a new screen manager.
func NewManager(dc *gg.Context, width, height int, updateFn func()) *Manager {
	return &Manager{
		dc:       dc,
		width:    width,
		height:   height,
		updateFn: updateFn,
		screens:  make(map[ScreenID]Screen),
		timers:   make(map[TimerID]*screenTimer),
	}
}

// SetUpdateRectFn sets the function for partial screen updates.
func (m *Manager) SetUpdateRectFn(fn func(x, y, w, h int)) {
	m.updateRectFn = fn
}

// Register registers a screen with the manager.
func (m *Manager) Register(id ScreenID, screen Screen) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.screens[id] = screen
}

// SwitchTo transitions to a new screen.
func (m *Manager) SwitchTo(id ScreenID) {
	m.mu.Lock()

	screen, ok := m.screens[id]
	if !ok {
		m.mu.Unlock()
		log.Printf("Screen: unknown screen ID %d", id)
		return
	}

	if m.current != nil {
		// Clear all timers for the exiting screen
		m.clearTimersForScreenLocked(m.current)
		m.current.Exit()
	}

	m.current = screen
	m.mu.Unlock()

	// Call Init and Update outside the lock to allow SetTimeout to work
	screen.Init(m)

	// Check if we're still the current screen after Init (Init might have switched)
	m.mu.Lock()
	stillCurrent := (m.current == screen)
	m.mu.Unlock()

	if stillCurrent {
		screen.Update()
	}
}

// Current returns the current screen, or nil if none.
func (m *Manager) Current() Screen {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// SendEvent sends an event to the current screen.
func (m *Manager) SendEvent(event Event) bool {
	m.mu.Lock()
	current := m.current
	m.mu.Unlock()

	if current == nil {
		return false
	}
	// Call HandleEvent outside the lock to allow it to call Update/SwitchTo
	return current.HandleEvent(event)
}

// Update forces a redraw of the current screen.
func (m *Manager) Update() {
	m.mu.Lock()
	current := m.current
	m.mu.Unlock()

	if current != nil {
		current.Update()
	}
}

// DC returns the drawing context for screens to use.
func (m *Manager) DC() *gg.Context {
	return m.dc
}

// Width returns the screen width.
func (m *Manager) Width() int {
	return m.width
}

// Height returns the screen height.
func (m *Manager) Height() int {
	return m.height
}

// Flush flushes the drawing to the framebuffer.
func (m *Manager) Flush() {
	if m.updateFn != nil {
		m.updateFn()
	}
}

// FlushRect flushes only a rectangle of the screen to the framebuffer.
// Falls back to full flush if partial update is not supported.
func (m *Manager) FlushRect(x, y, w, h int) {
	if m.updateRectFn != nil {
		m.updateRectFn(x, y, w, h)
	} else if m.updateFn != nil {
		m.updateFn()
	}
}

// FillRect fills a rectangle with a solid color.
func (m *Manager) FillRect(x, y, w, h int, r, g, b float64) {
	m.dc.SetRGB(r, g, b)
	m.dc.DrawRectangle(float64(x), float64(y), float64(w), float64(h))
	m.dc.Fill()
}

// SetFontSize loads a font at the specified size.
func (m *Manager) SetFontSize(size int) {
	fontPath := "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
	if err := m.dc.LoadFontFace(fontPath, float64(size)); err != nil {
		log.Printf("Screen: failed to load font: %v", err)
	}
}

// DrawCentered draws centered text at the given y position.
func (m *Manager) DrawCentered(text string, y float64, r, g, b float64) {
	m.dc.SetRGB(r, g, b)
	m.dc.DrawStringAnchored(text, float64(m.width/2), y, 0.5, 0.5)
}

// FillBackground fills the screen with a solid color.
func (m *Manager) FillBackground(r, g, b float64) {
	m.dc.SetRGB(r, g, b)
	m.dc.DrawRectangle(0, 0, float64(m.width), float64(m.height))
	m.dc.Fill()
}

// SetMQTTConnected updates the MQTT connection state.
// This is called by the app framework and persists across screen switches.
// If the current screen is showing, it also sends an event to update the display.
func (m *Manager) SetMQTTConnected(connected bool) {
	m.mu.Lock()
	m.mqttConnected = connected
	current := m.current
	m.mu.Unlock()

	// Send event to current screen so it can react in real-time
	if current != nil {
		var eventType EventType
		if connected {
			eventType = EventMQTTConnected
		} else {
			eventType = EventMQTTDisconnected
		}
		current.HandleEvent(Event{Type: eventType})
	}
}

// IsMQTTConnected returns the current MQTT connection state.
func (m *Manager) IsMQTTConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mqttConnected
}

// SetVendingSession sets the current vending session info.
func (m *Manager) SetVendingSession(member, nickname string, amount float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vendingMember = member
	m.vendingNickname = nickname
	m.vendingAmount = amount
	m.vendingBalance = 1.00 // Mock balance for testing
	m.vendingAddAmount = 0
}

// GetVendingSession returns the current vending session info.
func (m *Manager) GetVendingSession() (member, nickname string, amount float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.vendingMember, m.vendingNickname, m.vendingAmount
}

// GetVendingBalance returns the current account balance.
func (m *Manager) GetVendingBalance() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.vendingBalance
}

// SetVendingAddAmount sets the amount to add to account.
func (m *Manager) SetVendingAddAmount(addAmount float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vendingAddAmount = addAmount
}

// GetVendingAddAmount returns the amount to add to account.
func (m *Manager) GetVendingAddAmount() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.vendingAddAmount
}

// ClearVendingSession clears the vending session state.
func (m *Manager) ClearVendingSession() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vendingMember = ""
	m.vendingNickname = ""
	m.vendingAmount = 0
	m.vendingBalance = 0
	m.vendingAddAmount = 0
}

// SetTimeout sets a one-shot timer that calls the callback after the duration.
// The callback receives the screen that was current when the timer was set.
// Returns a TimerID that can be used to cancel the timer.
func (m *Manager) SetTimeout(d time.Duration, callback TimerCallback) TimerID {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextTimerID++
	id := m.nextTimerID
	screen := m.current

	st := &screenTimer{
		id:       id,
		screen:   screen,
		callback: callback,
	}

	st.timer = time.AfterFunc(d, func() {
		m.mu.Lock()
		// Check if timer still exists (wasn't cleared)
		if _, exists := m.timers[id]; !exists {
			m.mu.Unlock()
			return
		}
		delete(m.timers, id)
		m.mu.Unlock()

		// Call callback outside of lock
		if callback != nil {
			callback(screen)
		}
	})

	m.timers[id] = st
	return id
}

// ClearTimeout cancels a specific timer by ID.
// Returns true if the timer was found and cancelled.
func (m *Manager) ClearTimeout(id TimerID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	st, exists := m.timers[id]
	if !exists {
		return false
	}

	st.timer.Stop()
	delete(m.timers, id)
	return true
}

// ClearAllTimeouts cancels all timers for the current screen.
func (m *Manager) ClearAllTimeouts() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current != nil {
		m.clearTimersForScreenLocked(m.current)
	}
}

// clearTimersForScreenLocked clears all timers associated with a screen.
// Must be called with m.mu held.
func (m *Manager) clearTimersForScreenLocked(screen Screen) {
	for id, st := range m.timers {
		if st.screen == screen {
			st.timer.Stop()
			delete(m.timers, id)
		}
	}
}
