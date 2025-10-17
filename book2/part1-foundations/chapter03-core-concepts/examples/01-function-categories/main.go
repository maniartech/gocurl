// Example 1: Function Categories
// Demonstrates all six gocurl function categories and when to use each.

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("📚 GoCurl Function Categories Demonstration\n")

	ctx := context.Background()

	// Category 1: Basic Functions - Returns *http.Response
	category1_Basic(ctx)

	// Category 2: String Functions - Returns body as string
	fmt.Println()
	category2_String(ctx)

	// Category 3: Bytes Functions - Returns body as []byte
	fmt.Println()
	category3_Bytes(ctx)

	// Category 4: JSON Functions - Unmarshals to struct
	fmt.Println()
	category4_JSON(ctx)

	// Category 5: Download Functions - Saves to file
	fmt.Println()
	category5_Download(ctx)

	// Category 6: WithVars Functions - Explicit variable control
	fmt.Println()
	category6_WithVars(ctx)
}

func category1_Basic(ctx context.Context) {
	fmt.Println("1️⃣  Category 1: Basic Curl() - Manual body reading")
	fmt.Println("   Use when: Need raw response, HEAD requests, custom parsing")

	// HEAD request - don't need body, just check if exists
	resp, err := gocurl.Curl(ctx, "-I", "https://api.github.com/zen")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Status: %d\n", resp.StatusCode)
	fmt.Printf("   📋 Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("   ⚙️  Server: %s\n", resp.Header.Get("Server"))
}

func category2_String(ctx context.Context) {
	fmt.Println("2️⃣  Category 2: CurlString() - Automatic string reading")
	fmt.Println("   Use when: Response is text, want immediate string")

	body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/zen")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Status: %d\n", resp.StatusCode)
	fmt.Printf("   📝 Body (string): %s\n", body)
	fmt.Printf("   💡 Body is automatically read as string!\n")
}

func category3_Bytes(ctx context.Context) {
	fmt.Println("3️⃣  Category 3: CurlBytes() - Automatic bytes reading")
	fmt.Println("   Use when: Binary data, need []byte for processing")

	bodyBytes, resp, err := gocurl.CurlBytes(ctx, "https://api.github.com/zen")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Status: %d\n", resp.StatusCode)
	fmt.Printf("   📦 Body (bytes): %d bytes\n", len(bodyBytes))
	fmt.Printf("   💾 Can save to file: os.WriteFile(\"output.txt\", bodyBytes, 0644)\n")
}

func category4_JSON(ctx context.Context) {
	fmt.Println("4️⃣  Category 4: CurlJSON() - Automatic JSON unmarshaling")
	fmt.Println("   Use when: API returns JSON, have type definition")

	type HTTPBinResponse struct {
		Origin string            `json:"origin"`
		URL    string            `json:"url"`
		Args   map[string]string `json:"args"`
	}

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result, "https://httpbin.org/get")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Status: %d\n", resp.StatusCode)
	fmt.Printf("   🎯 Origin: %s\n", result.Origin)
	fmt.Printf("   🔗 URL: %s\n", result.URL)
	fmt.Printf("   💡 JSON automatically unmarshaled to struct!\n")
}

func category5_Download(ctx context.Context) {
	fmt.Println("5️⃣  Category 5: CurlDownload() - Stream to file")
	fmt.Println("   Use when: Large files, want to save directly")

	tmpFile := os.TempDir() + "/gocurl-test.json"
	written, resp, err := gocurl.CurlDownload(ctx, tmpFile, "https://httpbin.org/json")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Status: %d\n", resp.StatusCode)
	fmt.Printf("   💾 Downloaded: %d bytes\n", written)
	fmt.Printf("   📁 Saved to: %s\n", tmpFile)

	// Cleanup
	os.Remove(tmpFile)
	fmt.Printf("   🧹 Cleaned up temporary file\n")
}

func category6_WithVars(ctx context.Context) {
	fmt.Println("6️⃣  Category 6: WithVars() - Explicit variable control")
	fmt.Println("   Use when: Testing, need explicit vars, no env expansion")

	// Define variables explicitly (doesn't use environment)
	vars := gocurl.Variables{
		"endpoint": "/get",
		"param":    "test",
	}

	resp, err := gocurl.CurlCommandWithVars(ctx, vars,
		`curl https://httpbin.org${endpoint}?value=${param}`)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Status: %d\n", resp.StatusCode)
	fmt.Printf("   📝 Used variables: endpoint=%s, param=%s\n", vars["endpoint"], vars["param"])
	fmt.Printf("   💡 Variables expanded from map, not environment!\n")
	fmt.Printf("   � Note: WithVars returns *http.Response (read body manually if needed)\n")
}
