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
func (m *Multi) Granted() {
	for _, ind := range m.indicators {
		ind.Granted()
	}
}

// Denied implements Indicator.Denied.
func (m *Multi) Denied() {
	for _, ind := range m.indicators {
		ind.Denied()
	}
}

// Opening implements Indicator.Opening.
func (m *Multi) Opening() {
	for _, ind := range m.indicators {
		ind.Opening()
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
