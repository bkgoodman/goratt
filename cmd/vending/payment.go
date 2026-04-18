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

	fee := 0.0
	if addAmount > 0 && addAmount < 5.0 {
		fee = 0.30
	}

	// Create vending session for API
	session := &vending.VendingSession{
		Member:     member,
		Nickname:   nickname,
		Balance:    balance,
		Amount:     amount,
		AddAmount:  addAmount,
		ServiceFee: fee, // Added dynamic fee calculation
		LastLog:    lastLog,
	}

	log.Printf("Processing payment: Member=%s, Amount=$%.2f, AddAmount=$%.2f, Fee=$%.2f Balance=%2.f",
		session.Member, session.Amount, session.AddAmount, session.ServiceFee, session.Balance)

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
