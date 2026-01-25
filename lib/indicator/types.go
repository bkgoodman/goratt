package indicator

// AccessInfo contains information about an access attempt for display purposes.
type AccessInfo struct {
	Member   string
	Nickname string
	Warning  string
	Allowed  bool
}
