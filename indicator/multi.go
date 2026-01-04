package indicator

// Multi combines multiple Indicator implementations.
type Multi struct {
	indicators []Indicator
}

// Idle implements Indicator.Idle.
func (m *Multi) Idle() {
	for _, ind := range m.indicators {
		ind.Idle()
	}
}

// Granted implements Indicator.Granted.
func (m *Multi) Granted(info *AccessInfo) {
	for _, ind := range m.indicators {
		ind.Granted(info)
	}
}

// Denied implements Indicator.Denied.
func (m *Multi) Denied(info *AccessInfo) {
	for _, ind := range m.indicators {
		ind.Denied(info)
	}
}

// Opening implements Indicator.Opening.
func (m *Multi) Opening(info *AccessInfo) {
	for _, ind := range m.indicators {
		ind.Opening(info)
	}
}

// ConnectionLost implements Indicator.ConnectionLost.
func (m *Multi) ConnectionLost() {
	for _, ind := range m.indicators {
		ind.ConnectionLost()
	}
}

// Shutdown implements Indicator.Shutdown.
func (m *Multi) Shutdown() {
	for _, ind := range m.indicators {
		ind.Shutdown()
	}
}

// Release implements Indicator.Release.
func (m *Multi) Release() error {
	var lastErr error
	for _, ind := range m.indicators {
		if err := ind.Release(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
