package indicator

// Noop implements Indicator but does nothing.
// Used when no indicators are configured.
type Noop struct{}

// Idle implements Indicator.Idle.
func (n *Noop) Idle() {}

// Granted implements Indicator.Granted.
func (n *Noop) Granted() {}

// Denied implements Indicator.Denied.
func (n *Noop) Denied() {}

// Opening implements Indicator.Opening.
func (n *Noop) Opening() {}

// ConnectionLost implements Indicator.ConnectionLost.
func (n *Noop) ConnectionLost() {}

// Shutdown implements Indicator.Shutdown.
func (n *Noop) Shutdown() {}

// Release implements Indicator.Release.
func (n *Noop) Release() error {
	return nil
}
