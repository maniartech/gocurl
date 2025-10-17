package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

// calculateSHA256 computes the SHA256 hash of a file
func calculateSHA256(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return hashString, nil
}

// downloadWithIntegrityCheck downloads a file and verifies its checksum
func downloadWithIntegrityCheck(ctx context.Context, url, filepath, expectedHash string) error {
	fmt.Printf("📡 Downloading: %s\n", url)
	fmt.Printf("   Saving to: %s\n", filepath)

	// Download file
	written, resp, err := gocurl.CurlDownload(ctx, filepath, url)
	if err != nil {
		// Clean up partial file on error
		os.Remove(filepath)
		return fmt.Errorf("download failed: %w", err)
	}

	if resp.StatusCode != 200 {
		os.Remove(filepath)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	fmt.Printf("   Downloaded: %d bytes\n\n", written)

	// Calculate checksum
	fmt.Println("🔐 Verifying file integrity...")
	actualHash, err := calculateSHA256(filepath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	fmt.Printf("   Expected: %s\n", expectedHash)
	fmt.Printf("   Actual:   %s\n\n", actualHash)

	// Verify
	if actualHash != expectedHash {
		os.Remove(filepath)
		return fmt.Errorf("❌ integrity check FAILED - file corrupted")
	}

	fmt.Println("✅ Integrity check PASSED - file is valid")
	return nil
}

// streamingHashCalculation demonstrates calculating hash while downloading
func streamingHashCalculation(ctx context.Context, url, filepath string) (string, error) {
	fmt.Printf("📡 Downloading with streaming hash calculation...\n")

	// Make request
	resp, err := gocurl.Curl(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create hash
	hash := sha256.New()

	// Use MultiWriter to write to both file and hash
	multiWriter := io.MultiWriter(file, hash)

	// Copy (calculates hash while downloading)
	written, err := io.Copy(multiWriter, resp.Body)
	if err != nil {
		os.Remove(filepath)
		return "", err
	}

	hashString := hex.EncodeToString(hash.Sum(nil))

	fmt.Printf("   Downloaded: %d bytes\n", written)
	fmt.Printf("   SHA256: %s\n", hashString)

	return hashString, nil
}

func main() {
	fmt.Println("Example 8: File Integrity Verification")
	fmt.Println("=======================================\n")

	ctx := context.Background()

	// Test 1: Download with known checksum
	fmt.Println("Test 1: Download and Verify Checksum")
	fmt.Println(string(make([]byte, 45)) + "\n")

	// For this demo, we'll download a file twice
	// First download to get the hash
	url := "https://httpbin.org/bytes/10240" // 10KB
	tempFile := "test-integrity-temp.bin"

	fmt.Println("Step 1: Initial download to get expected hash...")
	written, resp, err := gocurl.CurlDownload(ctx, tempFile, url)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Status: %d, Downloaded: %d bytes\n", resp.StatusCode, written)

	expectedHash, err := calculateSHA256(tempFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Hash: %s\n\n", expectedHash)

	// Now demonstrate verification
	fmt.Println("Step 2: Download again and verify integrity...")
	verifyFile := "test-integrity-verify.bin"
	err = downloadWithIntegrityCheck(ctx, url, verifyFile, expectedHash)
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}

	// Test 2: Streaming hash calculation
	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("\nTest 2: Streaming Hash Calculation")
	fmt.Println(string(make([]byte, 45)) + "\n")

	streamFile := "test-integrity-stream.bin"
	hash, err := streamingHashCalculation(ctx, url, streamFile)
	if err != nil {
		log.Fatalf("Streaming hash failed: %v", err)
	}

	fmt.Printf("\n🔐 Hash matches: %v\n", hash == expectedHash)

	// Test 3: Demonstrate corruption detection
	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("\nTest 3: Detecting Corrupted File")
	fmt.Println(string(make([]byte, 45)) + "\n")

	corruptFile := "test-integrity-corrupt.bin"

	// Download file
	written, resp, err = gocurl.CurlDownload(ctx, url, corruptFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Downloaded: %d bytes\n", written)

	// Corrupt the file
	file, err := os.OpenFile(corruptFile, os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	file.WriteAt([]byte("CORRUPT"), 100) // Corrupt some bytes
	file.Close()

	fmt.Println("   Corrupted file artificially\n")

	// Try to verify (should fail)
	fmt.Println("🔐 Verifying corrupted file...")
	actualHash, _ := calculateSHA256(corruptFile)
	fmt.Printf("   Expected: %s\n", expectedHash)
	fmt.Printf("   Actual:   %s\n\n", actualHash)

	if actualHash != expectedHash {
		fmt.Println("❌ Corruption detected (as expected)!")
	}

	// Summary
	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("\nKey Features:")
	fmt.Println("  • SHA256 checksum verification")
	fmt.Println("  • Detect file corruption/tampering")
	fmt.Println("  • Streaming hash calculation (no extra I/O)")
	fmt.Println("  • Automatic cleanup on verification failure")
	fmt.Println("\nUse Cases:")
	fmt.Println("  • Software distribution (verify downloads)")
	fmt.Println("  • Security-sensitive transfers")
	fmt.Println("  • Backup verification")
	fmt.Println("  • Update systems")
	fmt.Println("\nBest Practices:")
	fmt.Println("  • Always verify checksums for important files")
	fmt.Println("  • Use HTTPS + checksums for security")
	fmt.Println("  • Clean up failed downloads")
	fmt.Println("  • Log verification results")
	fmt.Println("\nCleanup:")
	fmt.Printf("  rm %s %s %s %s\n", tempFile, verifyFile, streamFile, corruptFile)
}
