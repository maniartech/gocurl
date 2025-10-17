package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

// ProgressReader wraps an io.Reader and tracks progress
type ProgressReader struct {
	reader      io.Reader
	total       int64
	read        int64
	onProgress  func(read, total int64, percent int)
	lastPercent int
}

func NewProgressReader(reader io.Reader, total int64, onProgress func(int64, int64, int)) *ProgressReader {
	return &ProgressReader{
		reader:     reader,
		total:      total,
		onProgress: onProgress,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	// Calculate percentage
	percent := 0
	if pr.total > 0 {
		percent = int(float64(pr.read) / float64(pr.total) * 100)
	}

	// Call progress callback every 10%
	if pr.onProgress != nil && percent >= pr.lastPercent+10 {
		pr.onProgress(pr.read, pr.total, percent)
		pr.lastPercent = percent
	}

	return n, err
}

func downloadWithProgress(ctx context.Context, url, filepath string) error {
	fmt.Printf("Step 1: Getting file size...\n")

	// First, get content length with HEAD request
	headResp, err := gocurl.Curl(ctx, url, "-I")
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	defer headResp.Body.Close()

	contentLength := headResp.ContentLength
	if contentLength < 0 {
		fmt.Println("Warning: Content-Length not available")
		contentLength = 0
	} else {
		fmt.Printf("File size: %d bytes (%.2f MB)\n\n", contentLength, float64(contentLength)/(1024*1024))
	}

	fmt.Printf("Step 2: Downloading to %s...\n", filepath)

	// Now download with progress
	resp, err := gocurl.Curl(ctx, url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Wrap reader with progress tracking
	progressReader := NewProgressReader(resp.Body, contentLength, func(read, total int64, percent int) {
		if total > 0 {
			fmt.Printf("Progress: %d%% (%d/%d bytes)\n", percent, read, total)
		} else {
			fmt.Printf("Downloaded: %d bytes\n", read)
		}
	})

	// Copy with progress
	written, err := io.Copy(file, progressReader)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	fmt.Printf("\n✅ Download complete: %d bytes written\n", written)
	return nil
}

func main() {
	fmt.Println("Example 2: Download with Progress Tracking")
	fmt.Println("===========================================\n")

	ctx := context.Background()

	// Download a small test file for demonstration
	url := "https://httpbin.org/bytes/1024000" // ~1MB
	filepath := "test-download.bin"

	err := downloadWithProgress(ctx, url, filepath)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("\nKey Features:")
	fmt.Println("  • HEAD request first to get file size")
	fmt.Println("  • Custom ProgressReader wraps io.Reader")
	fmt.Println("  • Progress callback called every 10%")
	fmt.Println("  • Works with any file size")
}
