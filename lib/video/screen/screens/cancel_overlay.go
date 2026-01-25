//go:build screen

package screens

import (
	"image"
	"image/draw"
	"time"

	"goratt/lib/video/screen"
)

// CancelMode represents how the cancel overlay was triggered.
type CancelMode int

const (
	// CancelModeTimeout is used when cancellation is triggered by activity timeout.
	CancelModeTimeout CancelMode = iota
	// CancelModeHold is used when cancellation is triggered by button long-press.
	CancelModeHold
)

// CancelOverlayConfig holds configuration for the cancel overlay.
type CancelOverlayConfig struct {
	BarY            int           // Y position of the overlay (top of the bar area)
	BarHeight       int           // Total height of overlay area
	DurationHold    time.Duration // Cancel countdown duration for hold mode
	DurationTimeout time.Duration // Cancel countdown duration for timeout mode
	Title           string        // e.g. "Canceling"
	HelpTextHold    string        // e.g. "Release to abort" (for hold mode)
	HelpTextTimeout string        // e.g. "Press button to abort" (for timeout mode)
}

// DefaultCancelOverlayConfig returns a default configuration.
func DefaultCancelOverlayConfig(mgr *screen.Manager) CancelOverlayConfig {
	return CancelOverlayConfig{
		BarY:            mgr.Height() - 150, // Moved up
		BarHeight:       140,                // Taller for more whitespace
		DurationHold:    2 * time.Second,
		DurationTimeout: 10 * time.Second, // Long timeout for inactivity
		Title:           "Canceling",
		HelpTextHold:    "Release to abort",
		HelpTextTimeout: "Press button to abort",
	}
}

// CancelOverlay provides visual feedback during cancellation.
// It is a reusable component that screens can embed.
type CancelOverlay struct {
	mgr        *screen.Manager
	config     CancelOverlayConfig
	mode       CancelMode
	startTime  time.Time
	progress   float64 // 0.0 to 1.0
	timerID    screen.TimerID
	active     bool
	onComplete func()
	onAbort    func()
	duration   time.Duration // Current active duration

	// For micro dirty-rect updates - only update the changed portion of progress bar
	barX          int // Left edge of progress bar
	barWidth      int // Total width of progress bar
	progBarY      int // Y position of the progress bar itself
	progBarH      int // Height of progress bar
	lastProgressX int // Last X position we drew to (for incremental updates)

	// Background backup for restoration
	backup *image.RGBA
}

// NewCancelOverlay creates a new cancel overlay with the given configuration.
func NewCancelOverlay(mgr *screen.Manager, config CancelOverlayConfig) *CancelOverlay {
	// Calculate progress bar dimensions with more horizontal padding
	padding := 60
	barX := padding
	barWidth := mgr.Width() - 2*padding

	// Layout: Title at top, progress bar in middle, help text at bottom
	progBarY := config.BarY + 65
	progBarH := 25

	return &CancelOverlay{
		mgr:           mgr,
		config:        config,
		barX:          barX,
		barWidth:      barWidth,
		progBarY:      progBarY,
		progBarH:      progBarH,
		lastProgressX: barX,
	}
}

// Start begins the cancel countdown.
func (c *CancelOverlay) Start(mode CancelMode, onComplete, onAbort func()) {
	c.mode = mode
	if mode == CancelModeTimeout {
		c.duration = c.config.DurationTimeout
	} else {
		c.duration = c.config.DurationHold
	}
	c.startTime = time.Now()
	c.progress = 0.0
	c.active = true
	c.onComplete = onComplete
	c.onAbort = onAbort
	c.lastProgressX = c.barX

	// Backup the area where the overlay will be drawn
	rect := image.Rect(0, c.config.BarY, c.mgr.Width(), c.config.BarY+c.config.BarHeight)
	c.backup = image.NewRGBA(rect)
	if rgba, ok := c.mgr.DC().Image().(*image.RGBA); ok {
		draw.Draw(c.backup, rect, rgba, rect.Min, draw.Src)
	}

	// Draw initial state
	c.drawFull()

	// Start animation timer
	c.scheduleNextFrame()
}

// scheduleNextFrame schedules the next animation frame.
func (c *CancelOverlay) scheduleNextFrame() {
	frameInterval := 50 * time.Millisecond

	c.timerID = c.mgr.SetTimeout(frameInterval, func(scr screen.Screen) {
		if !c.active {
			return
		}

		// Calculate progress
		elapsed := time.Since(c.startTime)
		c.progress = float64(elapsed) / float64(c.duration)

		if c.progress >= 1.0 {
			// Complete
			c.progress = 1.0
			c.active = false
			c.timerID = 0
			if c.onComplete != nil {
				c.onComplete()
			}
			return
		}

		// Update just the progress bar portion
		c.drawProgressIncrement()

		// Schedule next frame
		c.scheduleNextFrame()
	})
}

// HandleEvent processes an event and aborts the cancellation if appropriate.
// Returns true if the event was handled (the cancellation was aborted).
func (c *CancelOverlay) HandleEvent(event screen.Event) bool {
	if !c.active {
		return false
	}

	switch event.Type {
	case screen.EventRotaryTurn, screen.EventRotaryPress:
		if c.mode == CancelModeTimeout {
			c.Abort()
			return true
		}
	case screen.EventButtonUp:
		if c.mode == CancelModeHold {
			c.Abort()
			return true
		}
	}

	// In hold mode, we swallow turn/press events while canceling
	if c.mode == CancelModeHold && (event.Type == screen.EventRotaryTurn || event.Type == screen.EventRotaryPress) {
		return true
	}

	return false
}

// Abort stops the cancel countdown and calls the abort callback.
func (c *CancelOverlay) Abort() {
	if !c.active {
		return
	}

	c.active = false
	if c.timerID != 0 {
		c.mgr.ClearTimeout(c.timerID)
		c.timerID = 0
	}

	if c.onAbort != nil {
		c.onAbort()
	}

	// Restore background
	c.restoreBackground()
}

// restoreBackground restores the backed-up pixels to the screen.
func (c *CancelOverlay) restoreBackground() {
	if c.backup == nil {
		return
	}

	if rgba, ok := c.mgr.DC().Image().(*image.RGBA); ok {
		rect := c.backup.Bounds()
		draw.Draw(rgba, rect, c.backup, rect.Min, draw.Src)
		c.mgr.FlushRect(rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy())
	}
	c.backup = nil
}

// IsActive returns true if the cancel overlay is currently running.
func (c *CancelOverlay) IsActive() bool {
	return c.active
}

// Stop stops the overlay without calling any callbacks.
func (c *CancelOverlay) Stop() {
	c.active = false
	if c.timerID != 0 {
		c.mgr.ClearTimeout(c.timerID)
		c.timerID = 0
	}
	c.restoreBackground()
}

// drawFull draws the entire overlay area.
func (c *CancelOverlay) drawFull() {
	dc := c.mgr.DC()

	// Rounded rectangle background (White)
	paddingX := 40
	paddingY := 10
	ovlX := float64(paddingX)
	ovlY := float64(c.config.BarY + paddingY)
	ovlW := float64(c.mgr.Width() - 2*paddingX)
	ovlH := float64(c.config.BarHeight - 2*paddingY)
	radius := 20.0

	dc.SetRGB(1, 1, 1)
	dc.DrawRoundedRectangle(ovlX, ovlY, ovlW, ovlH, radius)
	dc.Fill()

	// Border
	dc.SetRGB(0.8, 0.8, 0.8)
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(ovlX, ovlY, ovlW, ovlH, radius)
	dc.Stroke()

	// Title text (large, black)
	c.mgr.SetFontSize(32)
	dc.SetRGB(0, 0, 0)
	dc.DrawStringAnchored(c.config.Title, float64(c.mgr.Width()/2), ovlY+30, 0.5, 0.5)

	// Progress bar background (light gray)
	dc.SetRGB(0.9, 0.9, 0.9)
	dc.DrawRectangle(float64(c.barX), float64(c.progBarY), float64(c.barWidth), float64(c.progBarH))
	dc.Fill()

	// Progress bar border
	dc.SetRGB(0.7, 0.7, 0.7)
	dc.SetLineWidth(1)
	dc.DrawRectangle(float64(c.barX), float64(c.progBarY), float64(c.barWidth), float64(c.progBarH))
	dc.Stroke()

	// Help text (small, gray)
	c.mgr.SetFontSize(16)
	dc.SetRGB(0.4, 0.4, 0.4)
	helpText := c.config.HelpTextHold
	if c.mode == CancelModeTimeout {
		helpText = c.config.HelpTextTimeout
	}
	dc.DrawStringAnchored(helpText, float64(c.mgr.Width()/2), ovlY+ovlH-25, 0.5, 0.5)

	// Flush the entire overlay area
	c.mgr.FlushRect(0, c.config.BarY, c.mgr.Width(), c.config.BarHeight)
}

// drawProgressIncrement draws only the newly filled portion of the progress bar.
func (c *CancelOverlay) drawProgressIncrement() {
	dc := c.mgr.DC()

	// Calculate the new progress bar fill position
	fillWidth := int(float64(c.barWidth) * c.progress)
	newX := c.barX + fillWidth

	// Only draw the increment since last update
	if newX <= c.lastProgressX {
		return
	}

	incrementWidth := newX - c.lastProgressX
	if incrementWidth < 1 {
		incrementWidth = 1
	}

	// Draw the progress fill (red/orange gradient effect - just use red for simplicity)
	dc.SetRGB(0.8, 0.2, 0.1)
	dc.DrawRectangle(float64(c.lastProgressX), float64(c.progBarY+2), float64(incrementWidth), float64(c.progBarH-4))
	dc.Fill()

	// Flush only the updated portion
	c.mgr.FlushRect(c.lastProgressX, c.progBarY, incrementWidth+2, c.progBarH)

	c.lastProgressX = newX
}

// Draw draws the current state of the overlay.
// Call this from the screen's Update() if you want to redraw the overlay
// (e.g., after a full screen redraw).
func (c *CancelOverlay) Draw() {
	if !c.active {
		return
	}
	c.drawFull()
	// Redraw progress up to current point
	if c.progress > 0 {
		dc := c.mgr.DC()
		fillWidth := int(float64(c.barWidth) * c.progress)
		dc.SetRGB(0.8, 0.2, 0.1)
		dc.DrawRectangle(float64(c.barX), float64(c.progBarY+2), float64(fillWidth), float64(c.progBarH-4))
		dc.Fill()
		c.mgr.FlushRect(c.barX, c.progBarY, fillWidth, c.progBarH)
		c.lastProgressX = c.barX + fillWidth
	}
}
