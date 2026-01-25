package door

// Noop implements DoorOpener but does nothing.
// Used when no door control is configured.
type Noop struct{}

// Open implements DoorOpener.Open.
func (n *Noop) Open() error {
	return nil
}

// Close implements DoorOpener.Close.
func (n *Noop) Close() error {
	return nil
}

// Release implements DoorOpener.Release.
func (n *Noop) Release() error {
	return nil
}
