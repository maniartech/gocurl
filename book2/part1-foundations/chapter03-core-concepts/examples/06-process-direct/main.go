// Example 6: Different Request Methods
// Demonstrates various HTTP methods (GET, POST, PUT, DELETE, etc.)

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

type HTTPBinResponse struct {
	Args    map[string]string      `json:"args"`
	Headers map[string]string      `json:"headers"`
	JSON    map[string]interface{} `json:"json,omitempty"`
	Data    string                 `json:"data,omitempty"`
	URL     string                 `json:"url"`
	Method  string                 `json:"method,omitempty"`
}

func main() {
	fmt.Println("🔧 HTTP Methods Demonstration\n")

	// Method 1: GET request
	method1_GET()

	// Method 2: POST request
	fmt.Println()
	method2_POST()

	// Method 3: PUT request
	fmt.Println()
	method3_PUT()

	// Method 4: DELETE request
	fmt.Println()
	method4_DELETE()

	// Method 5: HEAD request
	fmt.Println()
	method5_HEAD()

	// Method 6: PATCH request
	fmt.Println()
	method6_PATCH()
}

func method1_GET() {
	fmt.Println("1️⃣  GET Request")

	ctx := context.Background()

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result, "https://httpbin.org/get?foo=bar")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ GET request successful\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   � URL: %s\n", result.URL)
	fmt.Printf("   📋 Query params: %v\n", result.Args)
}

func method2_POST() {
	fmt.Println("2️⃣  POST Request")

	ctx := context.Background()
	jsonData := `{"name":"Alice","email":"alice@example.com"}`

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result,
		`curl -X POST https://httpbin.org/post`,
		`-H "Content-Type: application/json"`,
		`-d '`+jsonData+`'`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ POST request successful\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   📍 URL: %s\n", result.URL)
	fmt.Printf("   📦 Posted data: %v\n", result.JSON)
}

func method3_PUT() {
	fmt.Println("3️⃣  PUT Request")

	ctx := context.Background()
	jsonData := `{"name":"Updated Name"}`

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result,
		`curl -X PUT https://httpbin.org/put`,
		`-H "Content-Type: application/json"`,
		`-d '`+jsonData+`'`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ PUT request successful\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   📍 URL: %s\n", result.URL)
	fmt.Printf("   📦 Updated data: %v\n", result.JSON)
}

func method4_DELETE() {
	fmt.Println("4️⃣  DELETE Request")

	ctx := context.Background()

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result, `curl -X DELETE https://httpbin.org/delete`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ DELETE request successful\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   📍 URL: %s\n", result.URL)
}

func method5_HEAD() {
	fmt.Println("5️⃣  HEAD Request (Headers Only)")

	ctx := context.Background()

	// HEAD request only returns headers, no body
	resp, err := gocurl.Curl(ctx, `curl -I https://httpbin.org/get`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ HEAD request successful\n")
	fmt.Printf("   � Status: %d\n", resp.StatusCode)
	fmt.Printf("   📋 Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("   📏 Content-Length: %s\n", resp.Header.Get("Content-Length"))
	fmt.Printf("   🔧 Server: %s\n", resp.Header.Get("Server"))
	fmt.Printf("   💡 HEAD returns headers without body (faster)\n")
}

func method6_PATCH() {
	fmt.Println("6️⃣  PATCH Request")

	ctx := context.Background()
	patchData := `{"bio":"Updated bio"}`

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result,
		`curl -X PATCH https://httpbin.org/patch`,
		`-H "Content-Type: application/json"`,
		`-d '`+patchData+`'`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ PATCH request successful\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   📍 URL: %s\n", result.URL)
	fmt.Printf("   � Patched data: %v\n", result.JSON)
	fmt.Printf("\n   💡 HTTP Methods Summary:\n")
	fmt.Printf("      GET    - Retrieve data\n")
	fmt.Printf("      POST   - Create new resource\n")
	fmt.Printf("      PUT    - Update/replace resource\n")
	fmt.Printf("      PATCH  - Partial update\n")
	fmt.Printf("      DELETE - Remove resource\n")
	fmt.Printf("      HEAD   - Get headers only\n")
}
