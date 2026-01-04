package indicator

// Noop implements Indicator but does nothing.
// Used when no indicators are configured.
type Noop struct{}

// Idle implements Indicator.Idle.
func (n *Noop) Idle() {}

// Granted implements Indicator.Granted.
func (n *Noop) Granted(info *AccessInfo) {}

// Denied implements Indicator.Denied.
func (n *Noop) Denied(info *AccessInfo) {}

// Opening implements Indicator.Opening.
func (n *Noop) Opening(info *AccessInfo) {}

// ConnectionLost implements Indicator.ConnectionLost.
func (n *Noop) ConnectionLost() {}

// Shutdown implements Indicator.Shutdown.
func (n *Noop) Shutdown() {}

// Release implements Indicator.Release.
func (n *Noop) Release() error {
	return nil
}
