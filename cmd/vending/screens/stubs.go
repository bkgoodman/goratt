//go:build !screen
 
package vendingscreens
 
import "goratt/lib/video/screen"
 
type VendingIdleScreen struct{}
func NewVendingIdleScreen() *VendingIdleScreen { return &VendingIdleScreen{} }
func (s *VendingIdleScreen) SetBuildID(id string) {}
func (s *VendingIdleScreen) Init(mgr *screen.Manager) {}
func (s *VendingIdleScreen) Update() {}
func (s *VendingIdleScreen) HandleEvent(event screen.Event) bool { return false }
func (s *VendingIdleScreen) Exit() {}
func (s *VendingIdleScreen) Name() string { return "Idle" }
 
type VendingDeniedScreen struct{}
func NewVendingDeniedScreen() *VendingDeniedScreen { return &VendingDeniedScreen{} }
func (s *VendingDeniedScreen) SetInfo(member, nickname, warning string) {}
func (s *VendingDeniedScreen) Init(mgr *screen.Manager) {}
func (s *VendingDeniedScreen) Update() {}
func (s *VendingDeniedScreen) HandleEvent(event screen.Event) bool { return false }
func (s *VendingDeniedScreen) Exit() {}
func (s *VendingDeniedScreen) Name() string { return "Denied" }
 
type SelectAmountScreen struct{}
func NewSelectAmountScreen() *SelectAmountScreen { return &SelectAmountScreen{} }
func (s *SelectAmountScreen) Init(mgr *screen.Manager) {}
func (s *SelectAmountScreen) Update() {}
func (s *SelectAmountScreen) HandleEvent(event screen.Event) bool { return false }
func (s *SelectAmountScreen) Exit() {}
func (s *SelectAmountScreen) Name() string { return "SelectAmount" }
 
type ConfirmScreen struct{}
func NewConfirmScreen() *ConfirmScreen { return &ConfirmScreen{} }
func (s *ConfirmScreen) Init(mgr *screen.Manager) {}
func (s *ConfirmScreen) Update() {}
func (s *ConfirmScreen) HandleEvent(event screen.Event) bool { return false }
func (s *ConfirmScreen) Exit() {}
func (s *ConfirmScreen) Name() string { return "Confirm" }
 
type AbortedScreen struct{}
func NewAbortedScreen() *AbortedScreen { return &AbortedScreen{} }
func (s *AbortedScreen) Init(mgr *screen.Manager) {}
func (s *AbortedScreen) Update() {}
func (s *AbortedScreen) HandleEvent(event screen.Event) bool { return false }
func (s *AbortedScreen) Exit() {}
func (s *AbortedScreen) Name() string { return "Aborted" }
 
type InsufficientFundsScreen struct{}
func NewInsufficientFundsScreen() *InsufficientFundsScreen { return &InsufficientFundsScreen{} }
func (s *InsufficientFundsScreen) Init(mgr *screen.Manager) {}
func (s *InsufficientFundsScreen) Update() {}
func (s *InsufficientFundsScreen) HandleEvent(event screen.Event) bool { return false }
func (s *InsufficientFundsScreen) Exit() {}
func (s *InsufficientFundsScreen) Name() string { return "InsufficientFunds" }
 
type ProcessingScreen struct{}
func NewProcessingScreen() *ProcessingScreen { return &ProcessingScreen{} }
func (s *ProcessingScreen) Init(mgr *screen.Manager) {}
func (s *ProcessingScreen) Update() {}
func (s *ProcessingScreen) HandleEvent(event screen.Event) bool { return false }
func (s *ProcessingScreen) Exit() {}
func (s *ProcessingScreen) Name() string { return "Processing" }
 
type SuccessScreen struct{}
func NewSuccessScreen() *SuccessScreen { return &SuccessScreen{} }
func (s *SuccessScreen) Init(mgr *screen.Manager) {}
func (s *SuccessScreen) Update() {}
func (s *SuccessScreen) HandleEvent(event screen.Event) bool { return false }
func (s *SuccessScreen) Exit() {}
func (s *SuccessScreen) Name() string { return "Success" }
 
type PaymentFailedScreen struct{}
func NewPaymentFailedScreen() *PaymentFailedScreen { return &PaymentFailedScreen{} }
func (s *PaymentFailedScreen) Init(mgr *screen.Manager) {}
func (s *PaymentFailedScreen) Update() {}
func (s *PaymentFailedScreen) HandleEvent(event screen.Event) bool { return false }
func (s *PaymentFailedScreen) Exit() {}
func (s *PaymentFailedScreen) Name() string { return "PaymentFailed" }
