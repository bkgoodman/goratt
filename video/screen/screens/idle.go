//go:build screen

package screens

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"net"
	"strings"
	"time"

	"goratt/video/screen"
)

// startupTime is set once when the package is loaded
var startupTime = time.Now()

// Spinner pre-rendered frames (shared across instances)
var spinnerFrames []*image.RGBA
var spinnerSize = 40

func init() {
	// Pre-render 8 spinner frames
	spinnerFrames = make([]*image.RGBA, 8)
	for i := 0; i < 8; i++ {
		spinnerFrames[i] = renderSpinnerFrame(spinnerSize, i)
	}
}

// renderSpinnerFrame creates a single frame of the spinner animation
func renderSpinnerFrame(size, frame int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	cx, cy := float64(size)/2, float64(size)/2
	radius := float64(size)/2 - 4

	// Draw 8 dots around the circle, with varying brightness
	for dot := 0; dot < 8; dot++ {
		angle := float64(dot) * math.Pi / 4
		x := cx + radius*math.Cos(angle)
		y := cy + radius*math.Sin(angle)

		// Calculate brightness based on distance from current frame
		dist := (dot - frame + 8) % 8
		brightness := uint8(255 - dist*28) // Fade from 255 to ~59

		// Draw a filled circle (5x5 pixels for 40px spinner)
		dotRadius := size / 10 // 4 pixels for 40px spinner
		for dy := -dotRadius; dy <= dotRadius; dy++ {
			for dx := -dotRadius; dx <= dotRadius; dx++ {
				if dx*dx+dy*dy <= dotRadius*dotRadius {
					px, py := int(x)+dx, int(y)+dy
					if px >= 0 && px < size && py >= 0 && py < size {
						img.Set(px, py, color.RGBA{brightness, brightness, brightness, 255})
					}
				}
			}
		}
	}

	return img
}

// IdleScreen displays the ready/idle state.
type IdleScreen struct {
	mgr         *screen.Manager
	counter     int
	showCounter bool
	timerID     screen.TimerID

	// Counter display area for partial updates
	counterX      int
	counterY      int
	counterWidth  int
	counterHeight int

	// IP address display
	lastIP      string
	ipTimerID   screen.TimerID
	ipBarHeight int
}

// NewIdleScreen creates a new idle screen.
func NewIdleScreen() *IdleScreen {
	return &IdleScreen{
		ipBarHeight: 24,
	}
}

// getIPAddress returns the IP address of the primary network interface (wlan or eth).
func getIPAddress() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		// Skip loopback, down interfaces, and non-physical interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// Only consider wlan* or eth* interfaces
		if !strings.HasPrefix(iface.Name, "wlan") && !strings.HasPrefix(iface.Name, "eth") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip IPv6 and return first IPv4
			if ip == nil || ip.To4() == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

// shouldShowIP returns true if we're within 2 minutes of startup
func shouldShowIP() bool {
	return time.Since(startupTime) < 2*time.Minute
}

func (s *IdleScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	s.counter = 0
	s.showCounter = false
	s.timerID = 0

	// Calculate counter area - centered box for the number
	// Assuming max ~6 digits at 48pt font, roughly 120px wide
	s.counterWidth = 150
	s.counterX = (mgr.Width() - s.counterWidth) / 2
	s.counterY = mgr.Height()/2 + 10
	s.counterHeight = 60

	// Start IP address refresh timer if within startup window
	if shouldShowIP() {
		s.lastIP = getIPAddress()
		s.startIPRefresh()
	}
}

// startIPRefresh sets up a timer to periodically refresh the IP address display
func (s *IdleScreen) startIPRefresh() {
	if !shouldShowIP() {
		// Past the 2-minute window, stop refreshing
		s.ipTimerID = 0
		return
	}

	// Refresh every 5 seconds
	s.ipTimerID = s.mgr.SetTimeout(5*time.Second, func(scr screen.Screen) {
		if !shouldShowIP() {
			// Time's up, do a full redraw to remove the IP bar
			s.ipTimerID = 0
			s.mgr.Update()
			return
		}

		newIP := getIPAddress()
		if newIP != s.lastIP {
			s.lastIP = newIP
			s.updateIPBar()
		}
		s.startIPRefresh() // Schedule next refresh
	})
}

// updateIPBar does a partial update of just the IP address bar
func (s *IdleScreen) updateIPBar() {
	s.drawIPBar()
	s.mgr.FlushRect(0, 0, s.mgr.Width(), s.ipBarHeight)
}

// drawIPBar draws the IP address bar (call before Flush)
func (s *IdleScreen) drawIPBar() {
	if !shouldShowIP() {
		return
	}

	// White bar at top
	s.mgr.FillRect(0, 0, s.mgr.Width(), s.ipBarHeight, 1, 1, 1)

	// Black text with IP address
	s.mgr.SetFontSize(16)
	ip := s.lastIP
	if ip == "" {
		ip = "No IP"
	}
	s.mgr.DC().SetRGB(0, 0, 0)
	s.mgr.DC().DrawStringAnchored(ip, float64(s.mgr.Width()/2), float64(s.ipBarHeight/2), 0.5, 0.5)
}

func (s *IdleScreen) Update() {
	s.mgr.FillBackground(0, 0.5, 0) // Green background

	// Title
	s.mgr.SetFontSize(56)
	s.mgr.DrawCentered("Pay-by-RATT", float64(s.mgr.Height()/2)-40, 1, 1, 1)

	// Instruction
	s.mgr.SetFontSize(24)
	s.mgr.DrawCentered("Swipe key fob to begin", float64(s.mgr.Height()/2)+10, 0.9, 0.9, 0.9)

	// Show debug counter if active
	if s.showCounter {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(fmt.Sprintf("%d", s.counter), float64(s.mgr.Height()/2)+60, 1, 1, 0)
	}

	// Draw MQTT disconnected indicator if not connected
	s.drawMQTTIndicator()

	// Draw IP bar if within startup window
	s.drawIPBar()

	s.mgr.Flush()
}

// drawMQTTIndicator draws a red "NO MQTT" indicator at bottom if disconnected
func (s *IdleScreen) drawMQTTIndicator() {
	if s.mgr.IsMQTTConnected() {
		return
	}

	// Red bar at bottom
	barHeight := 24
	barY := s.mgr.Height() - barHeight
	s.mgr.FillRect(0, barY, s.mgr.Width(), barHeight, 0.8, 0, 0)

	// White text
	s.mgr.SetFontSize(16)
	s.mgr.DC().SetRGB(1, 1, 1)
	s.mgr.DC().DrawStringAnchored("NO MQTT", float64(s.mgr.Width()/2), float64(barY+barHeight/2), 0.5, 0.5)
}

// updateCounter does a partial update of just the counter area
func (s *IdleScreen) updateCounter() {
	// Clear the counter area with background color
	s.mgr.FillRect(s.counterX, s.counterY, s.counterWidth, s.counterHeight, 0, 0.5, 0)

	// Draw counter if visible
	if s.showCounter {
		s.mgr.SetFontSize(48)
		s.mgr.DrawCentered(fmt.Sprintf("%d", s.counter), float64(s.mgr.Height()/2)+40, 1, 1, 0)
	}

	// Flush only the counter area
	s.mgr.FlushRect(s.counterX, s.counterY, s.counterWidth, s.counterHeight)
}

func (s *IdleScreen) HandleEvent(event screen.Event) bool {
	switch event.Type {
	case screen.EventRotaryTurn:
		if rotary := event.Rotary(); rotary != nil {
			s.counter += rotary.Delta
			s.showCounter = true
			s.resetTimeout()
			s.updateCounter()
			return true
		}
	case screen.EventRotaryPress:
		if s.showCounter {
			s.showCounter = false
			s.counter = 0
			if s.timerID != 0 {
				s.mgr.ClearTimeout(s.timerID)
				s.timerID = 0
			}
			s.updateCounter()
			return true
		}
	case screen.EventMQTTConnected:
		s.updateMQTTIndicator()
		return true
	case screen.EventMQTTDisconnected:
		s.updateMQTTIndicator()
		return true
	}
	return false
}

// updateMQTTIndicator does a partial update of just the MQTT indicator area at bottom
func (s *IdleScreen) updateMQTTIndicator() {
	barHeight := 24
	barY := s.mgr.Height() - barHeight

	if s.mgr.IsMQTTConnected() {
		// Clear the bar area with background color
		s.mgr.FillRect(0, barY, s.mgr.Width(), barHeight, 0, 0.5, 0)
	} else {
		// Draw the red indicator
		s.drawMQTTIndicator()
	}

	s.mgr.FlushRect(0, barY, s.mgr.Width(), barHeight)
}

func (s *IdleScreen) resetTimeout() {
	// Clear existing timer
	if s.timerID != 0 {
		s.mgr.ClearTimeout(s.timerID)
	}
	// Set new 10 second timeout to hide counter
	s.timerID = s.mgr.SetTimeout(10*time.Second, func(scr screen.Screen) {
		s.showCounter = false
		s.counter = 0
		s.timerID = 0
		s.updateCounter()
	})
}

func (s *IdleScreen) Exit() {
	s.showCounter = false
	s.counter = 0
	s.timerID = 0
	s.ipTimerID = 0
}

func (s *IdleScreen) Name() string {
	return "Idle"
}
