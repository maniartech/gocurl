// Package main demonstrates Stripe API integration using GoCurl.
// This example creates a payment intent for processing card payments.
//
// Prerequisites:
//   - Stripe account (https://stripe.com)
//   - Test API key (https://dashboard.stripe.com/test/apikeys)
//   - Set environment variable: export STRIPE_SECRET_KEY="sk_test_..."
//
// Run: go run stripe_payment.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

// PaymentIntent represents a Stripe payment intent
type PaymentIntent struct {
	ID            string `json:"id"`
	Amount        int    `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	ClientSecret  string `json:"client_secret"`
	Created       int64  `json:"created"`
	PaymentMethod string `json:"payment_method"`
}

func main() {
	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	if apiKey == "" {
		log.Fatal("STRIPE_SECRET_KEY environment variable not set")
	}

	var intent PaymentIntent
	resp, err := gocurl.CurlJSON(ctx, &intent,
		"-X", "POST",
		"-u", apiKey+":",
		"-d", "amount=2000",
		"-d", "currency=usd",
		"-d", "payment_method_types[]=card",
		"-d", "description=Order #12345",
		"https://api.stripe.com/v1/payment_intents")

	if err != nil {
		log.Fatalf("Payment creation failed: %v", err)
	}
	defer resp.Body.Close()

	// Check for API errors
	if resp.StatusCode != 200 {
		log.Fatalf("API returned status %d", resp.StatusCode)
	}

	// Display payment intent details
	fmt.Printf("✅ Payment Intent Created\n")
	fmt.Printf("ID: %s\n", intent.ID)
	fmt.Printf("Amount: $%.2f %s\n", float64(intent.Amount)/100, intent.Currency)
	fmt.Printf("Status: %s\n", intent.Status)
	fmt.Printf("Client Secret: %s\n", intent.ClientSecret)
	fmt.Println("\nNote: This is a test payment. Use this client secret in your frontend to complete the payment.")
}
