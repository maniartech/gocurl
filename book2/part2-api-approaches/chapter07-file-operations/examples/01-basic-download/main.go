package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("Example 1: Basic File Download with CurlDownload")
	fmt.Println("==================================================\n")

	ctx := context.Background()

	// Download a test file from httpbin.org
	url := "https://httpbin.org/image/png"
	filepath := "downloaded-image.png"

	fmt.Printf("Downloading from: %s\n", url)
	fmt.Printf("Saving to: %s\n\n", filepath)

	// Download file
	written, resp, err := gocurl.CurlDownload(ctx, filepath, url)

	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	// Display results
	fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("Downloaded: %d bytes\n", written)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		fmt.Printf("Content-Length: %s bytes\n", contentLength)
	}

	fmt.Printf("\n✅ File saved to: %s\n", filepath)
	fmt.Println("\nKey Features:")
	fmt.Println("  • CurlDownload returns (bytes, response, error)")
	fmt.Println("  • File created automatically")
	fmt.Println("  • Memory efficient - streams to disk")
	fmt.Println("  • No need to read response body manually")
}
