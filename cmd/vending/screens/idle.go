//go:build screen

package vendingscreens

import (
	"net"
	"strings"
	"time"

	"goratt/lib/video/screen"
)

// startupTime is set once when the package is loaded
var startupTime = time.Now()

type VendingIdleScreen struct {
	mgr     *screen.Manager
	buildID string

	// IP address display
	lastIP        string
	ipTimerID     screen.TimerID
	ipHideTimerID screen.TimerID // Timer to auto-hide forced IP display
	ipBarHeight   int
	forceShowIP   bool // Force IP display even after startup window
	forceHideIP   bool // Force IP to hide even during startup window
}

func NewVendingIdleScreen() *VendingIdleScreen {
	return &VendingIdleScreen{
		ipBarHeight: 44,
	}
}

func (s *VendingIdleScreen) SetBuildID(id string) {
	s.buildID = id
}

// getIPAddress returns the IP address of the primary network interface (wlan or eth).
func getIPAddress() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
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
			if ip != nil && ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return ""
}

func shouldShowIP() bool {
	return time.Since(startupTime) < 2*time.Minute
}

func (s *VendingIdleScreen) shouldShowIPForScreen() bool {
	if shouldShowIP() && !s.forceHideIP {
		return true
	}
	return s.forceShowIP
}

func (s *VendingIdleScreen) Init(mgr *screen.Manager) {
	s.mgr = mgr
	if s.shouldShowIPForScreen() {
		s.lastIP = getIPAddress()
		s.startIPRefresh()
	}
}

func (s *VendingIdleScreen) startIPRefresh() {
	if !s.shouldShowIPForScreen() {
		s.ipTimerID = 0
		return
	}

	s.ipTimerID = s.mgr.SetTimeout(5*time.Second, func(scr screen.Screen) {
		if !s.shouldShowIPForScreen() {
			s.ipTimerID = 0
			s.mgr.Update()
			return
		}

		newIP := getIPAddress()
		if newIP != s.lastIP {
			s.lastIP = newIP
			s.updateIPBar()
		}
		s.startIPRefresh()
	})
}

func (s *VendingIdleScreen) updateIPBar() {
	s.drawIPBar()
	s.mgr.FlushRect(0, 0, s.mgr.Width(), s.ipBarHeight)
}

func (s *VendingIdleScreen) drawIPBar() {
	if !s.shouldShowIPForScreen() {
		return
	}
	s.mgr.FillRect(0, 0, s.mgr.Width(), s.ipBarHeight, 1, 1, 1)
	s.mgr.DC().SetRGB(0, 0, 0)
	s.mgr.SetFontSize(16)

	ip := s.lastIP
	if ip == "" {
		ip = "No IP"
	}
	s.mgr.DC().DrawStringAnchored(ip, float64(s.mgr.Width()/2), 12, 0.5, 0.5)

	build := s.buildID
	if build == "" {
		build = "Unknown Build"
	}
	s.mgr.DC().DrawStringAnchored(build, float64(s.mgr.Width()/2), 32, 0.5, 0.5)
}

func (s *VendingIdleScreen) Update() {
	s.mgr.FillBackground(0, 0.4, 0.2) // Teak/Forest green

	s.mgr.SetFontSize(56)
	y := float64(s.mgr.Height()/2) - 40
	s.mgr.DrawCentered("Pay-by-RATT", y, 1, 1, 1)

	s.mgr.SetFontSize(24)
	s.mgr.DrawCentered("Swipe badge to buy", y+70, 0.9, 0.9, 0.9)
	s.mgr.DrawCentered("Charge to member's card-on-file", y+100, 0.9, 0.9, 0.9)

	s.drawIPBar()
	s.mgr.Flush()
}

func (s *VendingIdleScreen) HandleEvent(event screen.Event) bool {
	if event.Type == screen.EventRotaryLongPress {
		currentlyShowing := s.shouldShowIPForScreen()
		if currentlyShowing {
			if shouldShowIP() {
				s.forceHideIP = true
				s.forceShowIP = false
			} else {
				s.forceShowIP = false
			}
			if s.ipTimerID != 0 {
				s.mgr.ClearTimeout(s.ipTimerID)
				s.ipTimerID = 0
			}
			if s.ipHideTimerID != 0 {
				s.mgr.ClearTimeout(s.ipHideTimerID)
				s.ipHideTimerID = 0
			}
		} else {
			s.forceHideIP = false
			s.forceShowIP = true
			s.lastIP = getIPAddress()
			s.startIPRefresh()
			s.ipHideTimerID = s.mgr.SetTimeout(2*time.Minute, func(scr screen.Screen) {
				s.forceShowIP = false
				if s.ipTimerID != 0 {
					s.mgr.ClearTimeout(s.ipTimerID)
					s.ipTimerID = 0
				}
				s.ipHideTimerID = 0
				s.mgr.Update()
			})
		}
		s.mgr.Update()
		return true
	}
	return false
}

func (s *VendingIdleScreen) Exit() {
	s.ipTimerID = 0
	s.ipHideTimerID = 0
	s.forceShowIP = false
	s.forceHideIP = false
}

func (s *VendingIdleScreen) Name() string {
	return "VendingIdle"
}
