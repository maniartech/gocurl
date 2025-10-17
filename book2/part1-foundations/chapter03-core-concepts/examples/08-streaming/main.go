// Example 8: Streaming Large Responses
// Demonstrates efficient handling of large responses using streaming.

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("📡 Streaming Large Responses Demonstration\n")

	// Example 1: Download with progress
	example1_DownloadWithProgress()

	// Example 2: Stream line-by-line
	fmt.Println()
	example2_StreamLines()

	// Example 3: Buffered reading
	fmt.Println()
	example3_BufferedReading()
}

func example1_DownloadWithProgress() {
	fmt.Println("1️⃣  Download with Progress Tracking")

	ctx := context.Background()

	// Download a file with progress tracking
	// Using CurlDownload which streams to a file
	url := "https://httpbin.org/bytes/1048576" // 1 MB

	fmt.Println("   📥 Downloading 1 MB file...")
	bytesWritten, resp, err := gocurl.CurlDownload(ctx, "/tmp/download.bin", url)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Download complete\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   📏 Bytes written: %d\n", bytesWritten)
	fmt.Printf("   📁 File: /tmp/download.bin\n")
	fmt.Printf("\n   💡 CurlDownload streams directly to file\n")
	fmt.Printf("   💡 Memory efficient for large downloads\n")
}

func example2_StreamLines() {
	fmt.Println("2️⃣  Stream Line-by-Line Processing")

	ctx := context.Background()

	// Get response for streaming
	resp, err := gocurl.Curl(ctx, "https://httpbin.org/stream/5")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Println("   📡 Streaming lines...\n")

	// Create buffered scanner for line-by-line reading
	scanner := bufio.NewScanner(resp.Body)
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("   Line %d: %s...\n", lineNum, truncate(line, 60))
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
		return
	}

	fmt.Printf("\n   ✅ Processed %d lines\n", lineNum-1)
	fmt.Println("   💡 Scanner reads line-by-line without loading full response")
}

func example3_BufferedReading() {
	fmt.Println("3️⃣  Buffered Reading for Large Responses")

	ctx := context.Background()

	// Get large response
	resp, err := gocurl.Curl(ctx, "https://httpbin.org/bytes/10240") // 10 KB

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Println("   📡 Reading in chunks...\n")

	// Read in 1KB chunks
	buffer := make([]byte, 1024)
	totalRead := 0
	chunkNum := 1

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			totalRead += n
			fmt.Printf("   Chunk %d: %d bytes (total: %d)\n", chunkNum, n, totalRead)
			chunkNum++
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Read error: %v", err)
			return
		}
	}

	fmt.Printf("\n   ✅ Read %d bytes in %d chunks\n", totalRead, chunkNum-1)
	fmt.Println("   💡 Buffered reading controls memory usage")
	fmt.Println("   💡 Process data incrementally as it arrives")
}

// Helper function to truncate strings
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}
