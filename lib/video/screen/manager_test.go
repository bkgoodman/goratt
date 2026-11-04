//go:build screen

package screen

import (
	"testing"
	"time"

	"github.com/fogleman/gg"
)

type mockScreen struct {
	handledEvents []Event
	netDelta      int
}

func (s *mockScreen) Init(mgr *Manager) {}
func (s *mockScreen) Update()            {}
func (s *mockScreen) Exit()              {}
func (s *mockScreen) Name() string       { return "mock" }
func (s *mockScreen) HandleEvent(e Event) bool {
	s.handledEvents = append(s.handledEvents, e)
	if e.Type == EventRotaryTurn {
		if r := e.Rotary(); r != nil {
			s.netDelta += r.Delta
		}
	}
	return true
}

func TestRotaryDebounceMultipleEvents(t *testing.T) {
	dc := gg.NewContext(100, 100)
	mgr := NewManager(dc, 100, 100, func() {})
	mock := &mockScreen{}
	mgr.Register(ScreenIdle, mock)
	mgr.SwitchTo(ScreenIdle)

	// Simulate intentional turn
	mgr.SendEvent(Event{
		Type: EventRotaryTurn,
		Data: RotaryData{ID: RotaryMain, Delta: 1},
	})
	if mock.netDelta != 1 {
		t.Fatalf("expected netDelta 1, got %d", mock.netDelta)
	}

	// Wait past threshold (100ms)
	time.Sleep(120 * time.Millisecond)

	// Simulate two accidental wobble turns right before button press
	mgr.SendEvent(Event{
		Type: EventRotaryTurn,
		Data: RotaryData{ID: RotaryMain, Delta: 1},
	})
	mgr.SendEvent(Event{
		Type: EventRotaryTurn,
		Data: RotaryData{ID: RotaryMain, Delta: 1},
	})

	if mock.netDelta != 3 {
		t.Fatalf("expected netDelta 3 after wobbles, got %d", mock.netDelta)
	}

	// Button press arrives 10ms later
	time.Sleep(10 * time.Millisecond)
	pressTime := time.Now()
	mgr.SendEvent(Event{
		Type: EventRotaryPress,
		Data: RotaryData{ID: RotaryMain, PressedAt: pressTime},
	})

	// Both wobbles (+1, +1) within 100ms should be canceled by injecting -2
	// So netDelta should be back to 1 (the intentional turn)!
	if mock.netDelta != 1 {
		t.Fatalf("expected netDelta to be restored to 1, got %d", mock.netDelta)
	}
}

func TestRotaryDebounceBounceBack(t *testing.T) {
	dc := gg.NewContext(100, 100)
	mgr := NewManager(dc, 100, 100, func() {})
	mock := &mockScreen{}
	mgr.Register(ScreenIdle, mock)
	mgr.SwitchTo(ScreenIdle)

	// Simulate contact bounce: +1 then -1 (net 0) right before press
	mgr.SendEvent(Event{
		Type: EventRotaryTurn,
		Data: RotaryData{ID: RotaryMain, Delta: 1},
	})
	mgr.SendEvent(Event{
		Type: EventRotaryTurn,
		Data: RotaryData{ID: RotaryMain, Delta: -1},
	})

	if mock.netDelta != 0 {
		t.Fatalf("expected netDelta 0, got %d", mock.netDelta)
	}

	// Button press arrives
	pressTime := time.Now()
	mgr.SendEvent(Event{
		Type: EventRotaryPress,
		Data: RotaryData{ID: RotaryMain, PressedAt: pressTime},
	})

	// Since net delta in window was 0, nothing should be injected!
	if mock.netDelta != 0 {
		t.Fatalf("expected netDelta to remain 0, got %d", mock.netDelta)
	}
}

func TestRotarySuppressTurnAfterPress(t *testing.T) {
	dc := gg.NewContext(100, 100)
	mgr := NewManager(dc, 100, 100, func() {})
	mock := &mockScreen{}
	mgr.Register(ScreenIdle, mock)
	mgr.SwitchTo(ScreenIdle)

	// Button press
	mgr.SendEvent(Event{
		Type: EventRotaryPress,
		Data: RotaryData{ID: RotaryMain, PressedAt: time.Now()},
	})

	// Immediate turn event after press (shaft spring-back wobble, 20ms later)
	time.Sleep(20 * time.Millisecond)
	mgr.SendEvent(Event{
		Type: EventRotaryTurn,
		Data: RotaryData{ID: RotaryMain, Delta: 1},
	})

	// Should have been dropped completely
	if mock.netDelta != 0 {
		t.Fatalf("expected turn after press to be dropped (netDelta 0), got %d", mock.netDelta)
	}
}

