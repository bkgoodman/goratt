package vending

import (
	"fmt"
	"log"
)

// PaymentProcessor interface for screens to use
type PaymentProcessor interface {
	ProcessPayment() error
}

// Global payment processor instance
var globalProcessor PaymentProcessor

// SetGlobalProcessor sets the global payment processor
func SetGlobalProcessor(processor PaymentProcessor) {
	globalProcessor = processor
}

// GetGlobalProcessor returns the global payment processor
func GetGlobalProcessor() PaymentProcessor {
	return globalProcessor
}

// ProcessPayment calls the global payment processor
func ProcessPayment() error {
	if globalProcessor == nil {
		return fmt.Errorf("no payment processor available")
	}
	return globalProcessor.ProcessPayment()
}

// MockProcessor for testing
type MockProcessor struct {
	ShouldFail bool
}

func (m *MockProcessor) ProcessPayment() error {
	if m.ShouldFail {
		return fmt.Errorf("mock payment failure")
	}
	log.Printf("Mock payment processed successfully")
	return nil
}
