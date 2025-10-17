package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

// StreamLargeFile downloads and processes a large file in chunks
func streamLargeFile(ctx context.Context, url, outputPath string) error {
	fmt.Println("📡 Making request...")

	// Make request
	resp, err := gocurl.Curl(ctx, url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	contentLength := resp.ContentLength
	if contentLength > 0 {
		fmt.Printf("   File size: %d bytes (%.2f MB)\n",
			contentLength, float64(contentLength)/(1024*1024))
	} else {
		fmt.Println("   File size: Unknown")
	}

	// Create output file
	fmt.Printf("\n📝 Creating output file: %s\n", outputPath)
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Stream directly to file (memory efficient)
	fmt.Println("\n📥 Streaming to disk...")
	fmt.Println("   (This uses only 32KB of memory regardless of file size)")

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	fmt.Printf("\n✅ Streamed %d bytes to %s\n", written, outputPath)
	fmt.Printf("   Memory used: ~32KB (buffer size)\n")

	return nil
}

// ProcessLargeFileInChunks demonstrates processing file data in chunks
func processLargeFileInChunks(ctx context.Context, url string, chunkSize int) error {
	fmt.Println("\n📡 Making request for chunk processing...")

	resp, err := gocurl.Curl(ctx, url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	fmt.Printf("   Status: %d %s\n", resp.StatusCode, resp.Status)

	// Create buffered reader
	reader := bufio.NewReader(resp.Body)
	buffer := make([]byte, chunkSize)

	totalProcessed := 0
	chunkNum := 0

	fmt.Printf("\n📦 Processing in %d byte chunks...\n\n", chunkSize)

	for {
		// Read chunk
		n, err := reader.Read(buffer)
		if n > 0 {
			chunkNum++
			totalProcessed += n

			// Process chunk (e.g., compute hash, parse data, etc.)
			fmt.Printf("   Chunk %d: %d bytes (total: %d)\n",
				chunkNum, n, totalProcessed)

			// Your processing logic here
			// For example: hash.Write(buffer[:n])
			// Or parse the data, transform it, etc.
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read error: %w", err)
		}
	}

	fmt.Printf("\n✅ Processed %d chunks (%d total bytes)\n",
		chunkNum, totalProcessed)
	fmt.Printf("   Memory used: %d bytes (chunk buffer)\n", chunkSize)

	return nil
}

func createLargeTestFile(filepath string, sizeMB int) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write MB at a time
	data := make([]byte, 1024*1024) // 1MB
	for i := 0; i < sizeMB; i++ {
		// Fill with pattern
		for j := range data {
			data[j] = byte('A' + ((i + j) % 26))
		}
		if _, err := file.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	fmt.Println("Example 7: Efficient Large File Handling")
	fmt.Println("=========================================\n")

	ctx := context.Background()

	// Test 1: Stream to disk
	fmt.Println("Test 1: Streaming Large File to Disk")
	fmt.Println(string(make([]byte, 45)) + "\n")

	url := "https://httpbin.org/bytes/2097152" // 2MB
	outputFile := "large-download.bin"

	err := streamLargeFile(ctx, url, outputFile)
	if err != nil {
		log.Fatalf("Stream test failed: %v", err)
	}

	// Test 2: Process in chunks
	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("\nTest 2: Processing Large File in Chunks")
	fmt.Println(string(make([]byte, 45)) + "\n")

	chunkSize := 256 * 1024 // 256KB chunks
	err = processLargeFileInChunks(ctx, url, chunkSize)
	if err != nil {
		log.Fatalf("Chunk processing failed: %v", err)
	}

	// Summary
	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("\nKey Features:")
	fmt.Println("  • io.Copy uses 32KB internal buffer")
	fmt.Println("  • Can handle files of ANY size")
	fmt.Println("  • No memory allocation for file content")
	fmt.Println("  • Constant memory usage regardless of file size")
	fmt.Println("  • Chunk processing for data transformation")
	fmt.Println("\nMemory Efficiency:")
	fmt.Println("  • 10MB file:   ~32KB memory")
	fmt.Println("  • 100MB file:  ~32KB memory")
	fmt.Println("  • 1GB file:    ~32KB memory")
	fmt.Println("  • 10GB file:   ~32KB memory")
	fmt.Println("\nUse Cases:")
	fmt.Println("  • Download large software updates")
	fmt.Println("  • Stream video/audio files")
	fmt.Println("  • Process log files")
	fmt.Println("  • Download database dumps")
	fmt.Println("\nCleanup:")
	fmt.Printf("  rm %s\n", outputFile)
}
