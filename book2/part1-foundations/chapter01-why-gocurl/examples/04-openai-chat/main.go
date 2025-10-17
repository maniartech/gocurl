// Package main demonstrates OpenAI API integration using GoCurl.
// This example sends a chat completion request to GPT-4.
//
// Prerequisites:
//   - OpenAI API key (https://platform.openai.com/api-keys)
//   - Set environment variable: export OPENAI_API_KEY="sk-..."
//
// Run: go run openai_chat.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

// ChatResponse represents OpenAI chat completion response
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func main() {
	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Request payload
	payload := `{
		"model": "gpt-4",
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "Explain HTTP in one sentence."}
		],
		"max_tokens": 100
	}`

	var response ChatResponse
	resp, err := gocurl.CurlJSON(ctx, &response,
		"-X", "POST",
		"-H", "Authorization: Bearer "+apiKey,
		"-H", "Content-Type: application/json",
		"-d", payload,
		"https://api.openai.com/v1/chat/completions")

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check for API errors
	if resp.StatusCode != 200 {
		log.Fatalf("API returned status %d", resp.StatusCode)
	}

	// Display response
	if len(response.Choices) > 0 {
		fmt.Println("AI Response:", response.Choices[0].Message.Content)
		fmt.Printf("\nTokens used: %d (prompt: %d, completion: %d)\n",
			response.Usage.TotalTokens,
			response.Usage.PromptTokens,
			response.Usage.CompletionTokens)
	} else {
		log.Fatal("No response choices returned")
	}
}
