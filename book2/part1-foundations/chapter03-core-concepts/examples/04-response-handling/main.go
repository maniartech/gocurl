// Example 4: Response Handling
// Demonstrates different techniques for parsing and processing HTTP responses.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/maniartech/gocurl"
)

// Response structures
type HTTPBinResponse struct {
	Args    map[string]string      `json:"args"`
	Headers map[string]string      `json:"headers"`
	Origin  string                 `json:"origin"`
	URL     string                 `json:"url"`
	Data    string                 `json:"data,omitempty"`
	JSON    map[string]interface{} `json:"json,omitempty"`
}

type GitHubUser struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	AvatarURL string `json:"avatar_url"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	Blog      string `json:"blog"`
	Location  string `json:"location"`
	Bio       string `json:"bio"`
}

func main() {
	fmt.Println("📦 Response Handling Demonstration\n")

	// Technique 1: Automatic string reading
	technique1_AutomaticString()

	// Technique 2: Automatic bytes reading
	fmt.Println()
	technique2_AutomaticBytes()

	// Technique 3: Automatic JSON parsing
	fmt.Println()
	technique3_AutomaticJSON()

	// Technique 4: Manual response processing
	fmt.Println()
	technique4_ManualProcessing()

	// Technique 5: Header inspection
	fmt.Println()
	technique5_HeaderInspection()
}

func technique1_AutomaticString() {
	fmt.Println("1️⃣  Automatic String Reading (CurlString)")

	ctx := context.Background()
	body, resp, err := gocurl.CurlString(ctx, "https://httpbin.org/get?format=text")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Response read automatically\n")
	fmt.Printf("   📊 Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("   📏 Content length: %d bytes\n", len(body))
	fmt.Printf("   📝 Content type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("\n   💡 Body automatically read into string\n")
	fmt.Printf("   💡 No need to call ioutil.ReadAll()\n")
}

func technique2_AutomaticBytes() {
	fmt.Println("2️⃣  Automatic Bytes Reading (CurlBytes)")

	ctx := context.Background()
	data, resp, err := gocurl.CurlBytes(ctx, "https://httpbin.org/bytes/100")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Binary data read automatically\n")
	fmt.Printf("   📊 Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("   📏 Data size: %d bytes\n", len(data))
	fmt.Printf("   🔢 First 10 bytes: %v\n", data[:10])
	fmt.Printf("\n   💡 Body automatically read into []byte\n")
	fmt.Printf("   💡 Perfect for binary content (images, files, etc.)\n")
}

func technique3_AutomaticJSON() {
	fmt.Println("3️⃣  Automatic JSON Parsing (CurlJSON)")

	ctx := context.Background()
	var user GitHubUser

	resp, err := gocurl.CurlJSON(ctx, &user, "https://api.github.com/users/torvalds")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ JSON parsed automatically\n")
	fmt.Printf("   📊 Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("\n   👤 User Information:\n")
	fmt.Printf("      Login: %s\n", user.Login)
	fmt.Printf("      Name: %s\n", user.Name)
	fmt.Printf("      Location: %s\n", user.Location)
	fmt.Printf("      Bio: %s\n", user.Bio)
	fmt.Printf("\n   💡 JSON automatically unmarshaled into struct\n")
	fmt.Printf("   💡 No need to call json.Unmarshal()\n")
}

func technique4_ManualProcessing() {
	fmt.Println("4️⃣  Manual Response Processing (Curl)")

	ctx := context.Background()
	resp, err := gocurl.Curl(ctx, "https://httpbin.org/json")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Response received\n")
	fmt.Printf("   📊 Status: %d %s\n", resp.StatusCode, resp.Status)

	// Manual body reading
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		return
	}

	// Manual JSON parsing
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		return
	}

	fmt.Printf("   📦 Manual processing complete\n")
	fmt.Printf("   🗂️  JSON keys: %v\n", getKeys(result))
	fmt.Printf("\n   💡 Full control over response processing\n")
	fmt.Printf("   💡 Useful for streaming, custom parsing, etc.\n")
}

func technique5_HeaderInspection() {
	fmt.Println("5️⃣  Header Inspection")

	ctx := context.Background()
	resp, err := gocurl.Curl(ctx, "https://httpbin.org/response-headers?X-Custom-Header=CustomValue")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Headers retrieved\n")
	fmt.Printf("   📊 Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("\n   📋 Response Headers:\n")

	// Standard headers
	fmt.Printf("      Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("      Content-Length: %s\n", resp.Header.Get("Content-Length"))
	fmt.Printf("      Server: %s\n", resp.Header.Get("Server"))

	// Custom header
	if customHeader := resp.Header.Get("X-Custom-Header"); customHeader != "" {
		fmt.Printf("      X-Custom-Header: %s\n", customHeader)
	}

	// All headers
	fmt.Printf("\n   📦 All Headers (%d total):\n", len(resp.Header))
	for key, values := range resp.Header {
		fmt.Printf("      %s: %s\n", key, strings.Join(values, ", "))
	}

	fmt.Printf("\n   💡 Access any header via resp.Header.Get()\n")
	fmt.Printf("   💡 Use resp.Header for all headers\n")
}

// Helper function
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
