package main

import (
	"fmt"
	"log"

	"goratt/cmd/vending/vending"
)

// VendingSessionState holds the current vending session state
type VendingSessionState struct {
	Member  string
	Balance float64
	LastLog int
}

// ProcessPayment implements the PaymentProcessor interface
func (app *VendingApp) ProcessPayment() error {
	if app.vendingClient == nil {
		return fmt.Errorf("vending API not available")
	}

	if app.currentVendingSession == nil {
		return fmt.Errorf("no active vending session")
	}

	if app.Base.Display == nil {
		return fmt.Errorf("no display available")
	}

	mgr := app.Base.Display.Manager()
	member, nickname, amount := mgr.GetVendingSession()
	balance := mgr.GetVendingBalance()
	addAmount := mgr.GetVendingAddAmount()
	lastLog := mgr.GetVendingLastLog()

	// Create vending session for API
	session := &vending.VendingSession{
		Member:     member,
		Nickname:   nickname,
		Balance:    balance,
		Amount:     amount,
		AddAmount:  addAmount,
		ServiceFee: 0.30, // $0.30 service fee (could be configurable)
		LastLog:    lastLog,
	}

	log.Printf("Processing payment: Member=%s, Amount=$%.2f, AddAmount=$%.2f, Fee=$%.2f",
		session.Member, session.Amount, session.AddAmount, session.ServiceFee)

	// Process the payment
	if err := app.vendingClient.ProcessPurchase(session); err != nil {
		log.Printf("Payment processing failed: %v", err)
		return err
	}

	log.Printf("Payment processed successfully for %s", member)
	return nil
}

func (app *VendingApp) startVendingSession(member, nickname string, balance float64, lastLog int) {
	// Store balance and lastLog for payment processing
	if app.Base.Display != nil {
		mgr := app.Base.Display.Manager()
		mgr.SetVendingBalance(balance)
		mgr.SetVendingLastLog(lastLog)
		app.currentVendingSession = &VendingSessionState{
			Member:  member,
			Balance: balance,
			LastLog: lastLog,
		}
	}
}
