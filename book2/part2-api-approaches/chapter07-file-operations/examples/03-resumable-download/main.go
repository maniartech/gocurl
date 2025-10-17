package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/maniartech/gocurl"
)

func resumableDownload(ctx context.Context, url, filepath string) error {
	// Check if file exists and get its size
	var offset int64
	if info, err := os.Stat(filepath); err == nil {
		offset = info.Size()
		fmt.Printf("📁 Found existing file: %s (%d bytes)\n", filepath, offset)
		fmt.Printf("🔄 Resuming download from byte %d...\n\n", offset)
	} else {
		fmt.Printf("📝 Starting fresh download...\n\n")
	}

	// Open file for appending or creating
	flag := os.O_CREATE | os.O_WRONLY
	if offset > 0 {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(filepath, flag, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Make request with Range header
	fmt.Printf("📡 Requesting Range: bytes=%d-\n", offset)
	resp, err := gocurl.Curl(ctx, url,
		"-H", fmt.Sprintf("Range: bytes=%d-", offset))

	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check server response
	fmt.Printf("📥 Server Response: %d %s\n", resp.StatusCode, resp.Status)

	if resp.StatusCode == http.StatusPartialContent {
		fmt.Println("✅ Server supports range requests - resuming download")
		contentRange := resp.Header.Get("Content-Range")
		if contentRange != "" {
			fmt.Printf("   Content-Range: %s\n", contentRange)
		}
	} else if resp.StatusCode == http.StatusOK {
		fmt.Println("⚠️  Server does not support range requests")
		fmt.Println("   Starting download from beginning")
		// Truncate file and start over
		file.Truncate(0)
		file.Seek(0, 0)
		offset = 0
	} else {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Download remaining bytes
	fmt.Println("\n📦 Downloading...")
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("\n❌ Download interrupted: %v\n", err)
		return err
	}

	totalSize := offset + written
	fmt.Printf("\n✅ Downloaded %d bytes (previous: %d, new: %d)\n",
		totalSize, offset, written)
	fmt.Printf("   Total file size: %d bytes\n", totalSize)

	return nil
}

func main() {
	fmt.Println("Example 3: Resumable Downloads")
	fmt.Println("===============================\n")

	ctx := context.Background()

	// Use httpbin's streaming endpoint for testing
	url := "https://httpbin.org/bytes/500000" // 500KB
	filepath := "resumable-download.bin"

	fmt.Println("Demo: Simulating interrupted download")
	fmt.Println("--------------------------------------\n")

	// First attempt - download complete file
	fmt.Println("Attempt 1:")
	err := resumableDownload(ctx, url, filepath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\n" + string(make([]byte, 50)) + "\n")

	// Second attempt - simulates resume (will show existing file)
	fmt.Println("Attempt 2 (simulated resume):")
	err = resumableDownload(ctx, url, filepath)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("\n" + string(make([]byte, 50)) + "\n")
	fmt.Println("Key Features:")
	fmt.Println("  • Uses Range header to resume from offset")
	fmt.Println("  • Checks for 206 Partial Content response")
	fmt.Println("  • Falls back to full download if not supported")
	fmt.Println("  • Appends to existing file")
	fmt.Println("\nTo test real resumption:")
	fmt.Println("  1. Start download")
	fmt.Println("  2. Ctrl+C to interrupt")
	fmt.Println("  3. Run again - it will resume!")
}
