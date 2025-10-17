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

func uploadFile(ctx context.Context, url, fieldName, filepath string) error {
	fmt.Printf("📁 Opening file: %s\n", filepath)

	// Open file to upload
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fmt.Printf("   Size: %d bytes\n", fileInfo.Size())
	fmt.Printf("   Name: %s\n\n", fileInfo.Name())

	// Create multipart buffer
	fmt.Println("📦 Creating multipart form...")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file field
	part, err := writer.CreateFormFile(fieldName, fileInfo.Name())
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add other form fields
	writer.WriteField("description", "Uploaded via GoCurl")
	writer.WriteField("category", "test")
	writer.WriteField("timestamp", fmt.Sprintf("%d", fileInfo.ModTime().Unix()))

	// Close writer to finalize multipart message
	contentType := writer.FormDataContentType()
	writer.Close()

	fmt.Printf("   Content-Type: %s\n", contentType)
	fmt.Printf("   Total size: %d bytes\n\n", body.Len())

	// Build request
	fmt.Println("📡 Uploading to server...")
	opts := options.NewRequestOptionsBuilder().
		SetURL(url).
		SetMethod("POST").
		AddHeader("Content-Type", contentType).
		SetBody(body.String()).
		Build()

	// Execute
	httpResp, _, err := gocurl.Process(ctx, opts)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer httpResp.Body.Close()

	fmt.Printf("   Status: %d %s\n", httpResp.StatusCode, httpResp.Status)

	if httpResp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("server error: %s", string(bodyBytes))
	}

	// Read response
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("\n📥 Server Response:")
	fmt.Println(string(responseBody))

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
		data[i] = byte(i % 256)
	}

	_, err = file.Write(data)
	return err
}

func main() {
	fmt.Println("Example 5: File Upload with Multipart Form")
	fmt.Println("===========================================\n")

	ctx := context.Background()

	// Create a test file
	testFile := "test-upload.txt"
	fmt.Println("Creating test file...")
	err := createTestFile(testFile, 10) // 10KB
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	fmt.Printf("✅ Created %s\n\n", testFile)

	// Upload to httpbin.org
	url := "https://httpbin.org/post"

	fmt.Println("Starting upload...")
	fmt.Println(string(make([]byte, 50)) + "\n")

	err = uploadFile(ctx, url, "file", testFile)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("✅ Upload successful!\n")

	fmt.Println("Key Features:")
	fmt.Println("  • Multipart form-data encoding")
	fmt.Println("  • File uploaded with additional fields")
	fmt.Println("  • Automatic content-type handling")
	fmt.Println("  • Server echo response")
	fmt.Println("\nCleanup:")
	fmt.Printf("  rm %s\n", testFile)
}
