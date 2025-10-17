package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// ProgressWriter wraps io.Writer and tracks upload progress
type ProgressWriter struct {
	writer      io.Writer
	total       int64
	written     int64
	onProgress  func(written, total int64, percent int)
	lastPercent int
}

func NewProgressWriter(writer io.Writer, total int64, onProgress func(int64, int64, int)) *ProgressWriter {
	return &ProgressWriter{
		writer:     writer,
		total:      total,
		onProgress: onProgress,
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.written += int64(n)

	// Calculate percentage
	percent := 0
	if pw.total > 0 {
		percent = int(float64(pw.written) / float64(pw.total) * 100)
	}

	// Call progress callback every 10%
	if pw.onProgress != nil && percent >= pw.lastPercent+10 {
		pw.onProgress(pw.written, pw.total, percent)
		pw.lastPercent = percent
	}

	return n, err
}

func uploadWithProgress(ctx context.Context, url, fieldName, filepath string) error {
	// Open file
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fmt.Printf("📁 File: %s (%d bytes)\n\n", fileInfo.Name(), fileInfo.Size())

	// First pass: build multipart body to get size
	fmt.Println("📦 Preparing multipart form...")
	var tempBuffer bytes.Buffer
	tempWriter := multipart.NewWriter(&tempBuffer)

	part, err := tempWriter.CreateFormFile(fieldName, fileInfo.Name())
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	tempWriter.WriteField("description", "Upload with progress tracking")
	contentType := tempWriter.FormDataContentType()
	tempWriter.Close()

	totalSize := int64(tempBuffer.Len())
	fmt.Printf("   Total upload size: %d bytes\n\n", totalSize)

	// Create actual upload buffer with progress tracking
	fmt.Println("📡 Uploading with progress tracking...")
	var uploadBuffer bytes.Buffer

	progressWriter := NewProgressWriter(&uploadBuffer, totalSize, func(written, total int64, percent int) {
		fmt.Printf("Progress: %d%% (%d/%d bytes)\n", percent, written, total)
	})

	// Rebuild multipart with progress writer
	file.Seek(0, 0) // Reset file position

	writer := multipart.NewWriter(progressWriter)
	part, err = writer.CreateFormFile(fieldName, fileInfo.Name())
	if err != nil {
		return err
	}

	// This triggers progress callbacks
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	writer.WriteField("description", "Upload with progress tracking")
	writer.Close()

	fmt.Println("\n📤 Sending to server...")

	// Build and execute request
	opts := options.NewRequestOptionsBuilder().
		SetURL(url).
		SetMethod("POST").
		AddHeader("Content-Type", contentType).
		SetBody(uploadBuffer.String()).
		Build()

	httpResp, _, err := gocurl.Process(ctx, opts)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer httpResp.Body.Close()

	fmt.Printf("   Status: %d %s\n", httpResp.StatusCode, httpResp.Status)

	if httpResp.StatusCode >= 400 {
		return fmt.Errorf("server returned error: %d", httpResp.StatusCode)
	}

	fmt.Println("\n✅ Upload complete!")
	return nil
}

func createTestFile(filepath string, sizeKB int) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write test data
	data := make([]byte, sizeKB*1024)
	for i := range data {
		data[i] = byte('A' + (i % 26))
	}

	_, err = file.Write(data)
	return err
}

func main() {
	fmt.Println("Example 6: Upload with Progress Tracking")
	fmt.Println("=========================================\n")

	ctx := context.Background()

	// Create a larger test file
	testFile := "large-test-upload.txt"
	fileSizeKB := 500 // 500KB

	fmt.Printf("Creating test file (%d KB)...\n", fileSizeKB)
	err := createTestFile(testFile, fileSizeKB)
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	fmt.Printf("✅ Created %s\n\n", testFile)

	fmt.Println(string(make([]byte, 50)) + "\n")

	// Upload with progress
	url := "https://httpbin.org/post"
	err = uploadWithProgress(ctx, url, "file", testFile)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("\nKey Features:")
	fmt.Println("  • Custom ProgressWriter wraps io.Writer")
	fmt.Println("  • Progress callback every 10%")
	fmt.Println("  • Works with multipart forms")
	fmt.Println("  • Tracks upload progress in real-time")
	fmt.Println("\nNote:")
	fmt.Println("  Progress is tracked during multipart creation")
	fmt.Println("  Actual network upload happens after progress shows 100%")
	fmt.Println("\nCleanup:")
	fmt.Printf("  rm %s\n", testFile)
}
