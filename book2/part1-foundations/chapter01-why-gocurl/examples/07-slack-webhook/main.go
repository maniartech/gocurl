// Package main demonstrates Slack webhook integration using GoCurl.
// This example sends formatted messages to a Slack channel.
//
// Prerequisites:
//   - Slack workspace with admin access
//   - Create incoming webhook: https://api.slack.com/messaging/webhooks
//   - Set environment variable: export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
//
// Run: go run slack_webhook.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/maniartech/gocurl"
)

func main() {
	ctx := context.Background()

	// Get webhook URL from environment
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("SLACK_WEBHOOK_URL environment variable not set")
	}

	// Send simple text message
	sendSimpleMessage(ctx, webhookURL)

	// Wait a bit before sending next message
	time.Sleep(2 * time.Second)

	// Send rich formatted message
	sendRichMessage(ctx, webhookURL)
}

func sendSimpleMessage(ctx context.Context, webhookURL string) {
	payload := `{
		"text": "🚀 Deployment completed successfully!"
	}`

	resp, err := gocurl.Curl(ctx,
		"-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", payload,
		webhookURL)

	if err != nil {
		log.Printf("Failed to send simple message: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Println("✅ Simple message sent to Slack")
	} else {
		log.Printf("❌ Slack returned status %d", resp.StatusCode)
	}
}

func sendRichMessage(ctx context.Context, webhookURL string) {
	payload := `{
		"text": "Production Deployment Status",
		"blocks": [
			{
				"type": "header",
				"text": {
					"type": "plain_text",
					"text": "🚀 Production Deployment"
				}
			},
			{
				"type": "section",
				"fields": [
					{
						"type": "mrkdwn",
						"text": "*Status:*\n✅ Success"
					},
					{
						"type": "mrkdwn",
						"text": "*Version:*\nv1.2.3"
					},
					{
						"type": "mrkdwn",
						"text": "*Environment:*\nProduction"
					},
					{
						"type": "mrkdwn",
						"text": "*Duration:*\n45 seconds"
					}
				]
			},
			{
				"type": "section",
				"text": {
					"type": "mrkdwn",
					"text": "*Changes:*\n• Fixed payment processing bug\n• Updated API endpoints\n• Performance improvements"
				}
			},
			{
				"type": "divider"
			},
			{
				"type": "context",
				"elements": [
					{
						"type": "mrkdwn",
						"text": "Deployed by GitHub Actions | <https://github.com/org/repo|View Repository>"
					}
				]
			}
		]
	}`

	resp, err := gocurl.Curl(ctx,
		"-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", payload,
		webhookURL)

	if err != nil {
		log.Printf("Failed to send rich message: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Println("✅ Rich formatted message sent to Slack")
	} else {
		log.Printf("❌ Slack returned status %d", resp.StatusCode)
	}
}
