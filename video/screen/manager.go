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
type Manager struct {
	mu            sync.Mutex
	current       Screen
	screens       map[ScreenID]Screen
	dc            *gg.Context
	width, height int
	updateFn      func() // Called after drawing to flush to framebuffer

	// Timer management
	nextTimerID TimerID
	timers      map[TimerID]*screenTimer
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

// Register registers a screen with the manager.
func (m *Manager) Register(id ScreenID, screen Screen) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.screens[id] = screen
}

// SwitchTo transitions to a new screen.
func (m *Manager) SwitchTo(id ScreenID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	screen, ok := m.screens[id]
	if !ok {
		log.Printf("Screen: unknown screen ID %d", id)
		return
	}

	if m.current != nil {
		// Clear all timers for the exiting screen
		m.clearTimersForScreenLocked(m.current)
		m.current.Exit()
	}

	m.current = screen
	m.current.Init(m)
	m.current.Update()
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
	defer m.mu.Unlock()

	if m.current == nil {
		return false
	}
	return m.current.HandleEvent(event)
}

// Update forces a redraw of the current screen.
func (m *Manager) Update() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current != nil {
		m.current.Update()
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
