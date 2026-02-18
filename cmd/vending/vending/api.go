package vending

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client represents the vending API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	username   string
	password   string
	product    string
}

// NewClient creates a new vending API client
func NewClient(baseURL, username, password, product string) *Client {
	return &Client{
		BaseURL:  baseURL,
		username: username,
		password: password,
		product:  product,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// applyAuth sets HTTP Basic Auth if configured.
func (c *Client) applyAuth(req *http.Request) {
	if c.username == "" && c.password == "" {
		return
	}
	req.SetBasicAuth(c.username, c.password)
}

// BalanceResponse represents the response from queryBalance
type BalanceResponse struct {
	Status    string  `json:"status"`
	Balance   float64 `json:"balance,omitempty"`
	LastLog   int     `json:"lastLog,omitempty"`
	ErrorDesc string  `json:"description,omitempty"`
}

// ChargeRequest represents the request for chargeAccount
type ChargeRequest struct {
	Amount      int    `json:"amount"`      // Amount in cents
	PrevBalance int    `json:"prevBalance"` // Previous balance in cents
	LastLog     int    `json:"lastLog"`     // Last vending log ID
	Product     string `json:"product,omitempty"`
	Comment     string `json:"comment,omitempty"`
}

// ChargeResponse represents the response from chargeAccount
type ChargeResponse struct {
	Status    string `json:"status"`
	Member    string `json:"member,omitempty"`
	Customer  string `json:"customer,omitempty"`
	ErrorDesc string `json:"description,omitempty"`
}

// ReupRequest represents the request for reupBalance
type ReupRequest struct {
	AddAmount   int    `json:"addAmount"`   // Amount to add in cents
	TotalCharge int    `json:"totalCharge"` // Total charge in cents (addAmount + serviceFee)
	PrevBalance int    `json:"prevBalance"` // Previous balance in cents
	ServiceFee  int    `json:"serviceFee"`  // Service fee in cents
	PurchaseAmt int    `json:"purchaseAmt"` // Purchase amount in cents
	NewBalance  int    `json:"newBalance"`  // New balance in cents
	LastLog     int    `json:"lastLog"`     // Last vending log ID
	ProductCode string `json:"productCode,omitempty"`
	Description string `json:"description,omitempty"`
	Comment     string `json:"comment,omitempty"`
}

// ReupResponse represents the response from reupBalance
type ReupResponse struct {
	Status    string `json:"status"`
	Member    string `json:"member,omitempty"`
	Customer  string `json:"customer,omitempty"`
	ErrorDesc string `json:"description,omitempty"`
}

// QueryBalance queries the balance for a member
func (c *Client) QueryBalance(member string) (*BalanceResponse, error) {
	url := fmt.Sprintf("%s/api/v2/vending/queryBalance/%s", c.BaseURL, member)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.applyAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var balanceResp BalanceResponse
	if err := json.Unmarshal(body, &balanceResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if balanceResp.Status != "success" {
		return &balanceResp, fmt.Errorf("API error: %s", balanceResp.ErrorDesc)
	}

	return &balanceResp, nil
}

// ChargeAccount charges a member's account for a purchase
func (c *Client) ChargeAccount(member string, req ChargeRequest) (*ChargeResponse, error) {
	if c.product != "" {
		req.Product = c.product
	}
	url := fmt.Sprintf("%s/api/v2/vending/chargeAccount/%s", c.BaseURL, member)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.applyAuth(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var chargeResp ChargeResponse
	if err := json.Unmarshal(body, &chargeResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if chargeResp.Status != "success" {
		return &chargeResp, fmt.Errorf("API error: %s", chargeResp.ErrorDesc)
	}

	return &chargeResp, nil
}

// ReupBalance adds funds to a member's account and processes a purchase
func (c *Client) ReupBalance(member string, req ReupRequest) (*ReupResponse, error) {
	if c.product != "" {
		req.ProductCode = c.product
	}
	url := fmt.Sprintf("%s/api/v2/vending/reupBalance/%s", c.BaseURL, member)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.applyAuth(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var reupResp ReupResponse
	if err := json.Unmarshal(body, &reupResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if reupResp.Status != "success" {
		return &reupResp, fmt.Errorf("API error: %s", reupResp.ErrorDesc)
	}

	return &reupResp, nil
}

// DollarsToCents converts dollars to cents for API
func DollarsToCents(dollars float64) int {
	return int(dollars * 100)
}

// CentsToDollars converts cents from API to dollars
func CentsToDollars(cents int) float64 {
	return float64(cents) / 100.0
}

// VendingSession represents the current vending session state
type VendingSession struct {
	Member     string
	Nickname   string
	Balance    float64 // Current balance in dollars
	Amount     float64 // Purchase amount in dollars
	AddAmount  float64 // Amount to add in dollars
	ServiceFee float64 // Service fee in dollars
	LastLog    int     // Last vending log ID
}

// ProcessPurchase handles the complete purchase flow
func (c *Client) ProcessPurchase(session *VendingSession) error {
	// Convert to cents for API
	balanceCents := DollarsToCents(session.Balance)
	amountCents := DollarsToCents(session.Amount)
	addAmountCents := DollarsToCents(session.AddAmount)
	serviceFeeCents := DollarsToCents(session.ServiceFee)
	totalChargeCents := addAmountCents + serviceFeeCents
	newBalanceCents := balanceCents + addAmountCents - amountCents

	log.Printf("Processing purchase: Member=%s, Amount=$%.2f, AddAmount=$%.2f, Fee=$%.2f",
		session.Member, session.Amount, session.AddAmount, session.ServiceFee)

	if session.AddAmount > 0 {
		// Use reupBalance when adding funds
		req := ReupRequest{
			AddAmount:   addAmountCents,
			TotalCharge: totalChargeCents,
			PrevBalance: balanceCents,
			ServiceFee:  serviceFeeCents,
			PurchaseAmt: amountCents,
			NewBalance:  newBalanceCents,
			LastLog:     session.LastLog,
			Comment:     "Vending Purchase",
		}

		resp, err := c.ReupBalance(session.Member, req)
		if err != nil {
			return fmt.Errorf("reup balance failed: %w", err)
		}

		log.Printf("Reup successful: Member=%s, Customer=%s", resp.Member, resp.Customer)
	} else {
		// Use chargeAccount when just charging
		req := ChargeRequest{
			Amount:      amountCents,
			PrevBalance: balanceCents,
			LastLog:     session.LastLog,
			Comment:     "Vending Purchase",
		}

		resp, err := c.ChargeAccount(session.Member, req)
		if err != nil {
			return fmt.Errorf("charge account failed: %w", err)
		}

		log.Printf("Charge successful: Member=%s, Customer=%s", resp.Member, resp.Customer)
	}

	return nil
}
