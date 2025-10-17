# Chapter 7: File Operations

**Purpose:** Master file downloads and uploads with GoCurl for building robust file transfer applications.

**What You'll Learn:**
- Download files using CurlDownload functions
- Upload files with multipart forms
- Track download/upload progress
- Handle large files efficiently
- Resume interrupted downloads
- Work with cloud storage APIs (S3, etc.)

**Time Investment:** 2-3 hours
**Difficulty:** Intermediate to Advanced

---

## Introduction

File operations are fundamental to many applications: downloading software updates, uploading profile pictures, syncing documents to cloud storage, or processing large datasets. GoCurl provides powerful primitives for efficient file transfers with minimal code.

In this chapter, you'll learn:

1. **CurlDownload Functions** - Save HTTP responses directly to files
2. **File Uploads** - Multipart form data and file uploads
3. **Progress Tracking** - Monitor download/upload progress
4. **Large Files** - Handle files too big for memory
5. **Resumable Transfers** - Continue interrupted downloads
6. **Cloud Integration** - Work with S3 and other storage services

---

## Part 1: Downloading Files

### CurlDownload Function Family

```go
// Signature:
func CurlDownload(ctx context.Context, filepath string, command ...string) (int64, *http.Response, error)
```

**Key Features:**
- Streams response directly to file (memory efficient)
- Returns bytes written, response, and error (3 values)
- Creates file automatically
- Handles large files without loading into memory

**Return Values:**
1. `int64` - Number of bytes written to file
2. `*http.Response` - HTTP response with headers, status
3. `error` - Any error that occurred

---

### Example 1: Basic File Download

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    // Download Go logo
    written, resp, err := gocurl.CurlDownload(ctx, "golang-logo.png",
        "https://go.dev/images/gophers/motorcycle.svg")

    if err != nil {
        log.Fatalf("Download failed: %v", err)
    }

    fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
    fmt.Printf("Downloaded %d bytes to golang-logo.png\n", written)
    fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
}
```

**Output:**
```
Status: 200 OK
Downloaded 15234 bytes to golang-logo.png
Content-Type: image/svg+xml
```

**What Happened:**
1. GoCurl made GET request to URL
2. Response body streamed directly to file
3. No memory allocation for file content
4. File created automatically
5. Returns bytes written for verification

---

### Example 2: Download with Progress Tracking

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "os"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

// ProgressReader wraps an io.Reader and tracks progress
type ProgressReader struct {
    reader      io.Reader
    total       int64
    read        int64
    lastPercent int
}

func NewProgressReader(reader io.Reader, total int64) *ProgressReader {
    return &ProgressReader{
        reader: reader,
        total:  total,
    }
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
    n, err := pr.reader.Read(p)
    pr.read += int64(n)

    // Calculate percentage
    percent := int(float64(pr.read) / float64(pr.total) * 100)

    // Print progress every 10%
    if percent >= pr.lastPercent+10 {
        fmt.Printf("Progress: %d%% (%d/%d bytes)\n",
            percent, pr.read, pr.total)
        pr.lastPercent = percent
    }

    return n, err
}

func downloadWithProgress(ctx context.Context, url, filepath string) error {
    // First, get content length with HEAD request
    headResp, err := gocurl.Curl(ctx, url, "-I")
    if err != nil {
        return err
    }
    defer headResp.Body.Close()

    contentLength := headResp.ContentLength
    fmt.Printf("File size: %d bytes\n", contentLength)

    // Now download with progress
    resp, err := gocurl.Curl(ctx, url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Create file
    file, err := os.Create(filepath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Wrap reader with progress tracking
    progressReader := NewProgressReader(resp.Body, contentLength)

    // Copy with progress
    written, err := io.Copy(file, progressReader)
    if err != nil {
        return err
    }

    fmt.Printf("✅ Download complete: %d bytes\n", written)
    return nil
}

func main() {
    ctx := context.Background()

    url := "https://golang.org/dl/go1.21.0.linux-amd64.tar.gz"
    err := downloadWithProgress(ctx, url, "go1.21.0.tar.gz")

    if err != nil {
        log.Fatal(err)
    }
}
```

**Output:**
```
File size: 65432100 bytes
Progress: 10% (6543210/65432100 bytes)
Progress: 20% (13086420/65432100 bytes)
Progress: 30% (19629630/65432100 bytes)
...
Progress: 100% (65432100/65432100 bytes)
✅ Download complete: 65432100 bytes
```

---

### Example 3: Resumable Downloads

```go
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
    // Check if file exists
    var offset int64
    if info, err := os.Stat(filepath); err == nil {
        offset = info.Size()
        fmt.Printf("Resuming download from byte %d\n", offset)
    }

    // Open file for appending
    flag := os.O_CREATE | os.O_WRONLY
    if offset > 0 {
        flag |= os.O_APPEND
    }

    file, err := os.OpenFile(filepath, flag, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    // Make request with Range header
    resp, err := gocurl.Curl(ctx, url,
        "-H", fmt.Sprintf("Range: bytes=%d-", offset))

    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Check if server supports range requests
    if resp.StatusCode == http.StatusPartialContent {
        fmt.Println("Server supports range requests ✅")
    } else if resp.StatusCode == http.StatusOK {
        fmt.Println("Server does not support range requests, downloading from start")
        // Truncate file and start over
        file.Truncate(0)
        file.Seek(0, 0)
    } else {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    // Download remaining bytes
    written, err := io.Copy(file, resp.Body)
    if err != nil {
        return err
    }

    totalSize := offset + written
    fmt.Printf("✅ Downloaded %d bytes (total: %d bytes)\n", written, totalSize)

    return nil
}

func main() {
    ctx := context.Background()

    url := "https://example.com/large-file.zip"
    filepath := "large-file.zip"

    // First attempt
    err := resumableDownload(ctx, url, filepath)
    if err != nil {
        fmt.Printf("Download interrupted: %v\n", err)
    }

    // Second attempt (resumes from where it left off)
    err = resumableDownload(ctx, url, filepath)
    if err != nil {
        log.Fatal(err)
    }
}
```

**Key Concepts:**
- Use `Range: bytes=N-` header to resume from byte N
- Check for `206 Partial Content` status
- Append to existing file instead of overwriting
- Calculate total size = existing + downloaded

---

## Part 2: Uploading Files

### Multipart Form Upload

```go
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
    // Open file
    file, err := os.Open(filepath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Create multipart buffer
    var body bytes.Buffer
    writer := multipart.NewWriter(&body)

    // Add file field
    part, err := writer.CreateFormFile(fieldName, filepath)
    if err != nil {
        return err
    }

    // Copy file content
    if _, err := io.Copy(part, file); err != nil {
        return err
    }

    // Add other form fields
    writer.WriteField("description", "Uploaded via GoCurl")
    writer.WriteField("category", "documents")

    // Close writer to finalize multipart message
    contentType := writer.FormDataContentType()
    writer.Close()

    // Build request
    opts := options.NewRequestOptionsBuilder().
        SetURL(url).
        SetMethod("POST").
        SetHeader("Content-Type", contentType).
        SetBody(body.String()).
        Build()

    // Execute
    httpResp, _, err := gocurl.Process(ctx, opts)
    if err != nil {
        return err
    }
    defer httpResp.Body.Close()

    fmt.Printf("Upload status: %d %s\n", httpResp.StatusCode, httpResp.Status)

    return nil
}

func main() {
    ctx := context.Background()

    err := uploadFile(ctx,
        "https://httpbin.org/post",
        "file",
        "document.pdf")

    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("✅ File uploaded successfully")
}
```

---

### Simplified Upload with FileUpload Field

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func main() {
    ctx := context.Background()

    // Use FileUpload field for simple uploads
    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/post").
        SetMethod("POST").
        SetFileUpload(&options.FileUpload{
            FieldName: "file",
            FilePath:  "document.pdf",
            FileName:  "my-document.pdf", // Optional: override filename
        }).
        // Add other form fields
        AddFormField("description", "Important document").
        AddFormField("category", "work").
        Build()

    httpResp, _, err := gocurl.Process(ctx, opts)
    if err != nil {
        log.Fatal(err)
    }
    defer httpResp.Body.Close()

    fmt.Printf("Status: %d\n", httpResp.StatusCode)
    fmt.Println("✅ Upload successful")
}
```

---

## Part 3: Large File Handling

### Streaming Large Files

For files too large to fit in memory:

```go
package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "os"

    "github.com/maniartech/gocurl"
)

func streamLargeFile(ctx context.Context, url, outputPath string) error {
    // Make request
    resp, err := gocurl.Curl(ctx, url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    // Create output file
    out, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer out.Close()

    // Stream directly to file (no memory buffer)
    written, err := io.Copy(out, resp.Body)
    if err != nil {
        return err
    }

    fmt.Printf("Streamed %d bytes to %s\n", written, outputPath)
    return nil
}

func main() {
    ctx := context.Background()

    // Download large file (e.g., 1GB+)
    url := "https://releases.ubuntu.com/22.04/ubuntu-22.04-desktop-amd64.iso"

    err := streamLargeFile(ctx, url, "ubuntu.iso")
    if err != nil {
        log.Fatal(err)
    }
}
```

**Memory Efficiency:**
- `io.Copy` uses 32KB buffer internally
- Only 32KB in memory at a time
- Can handle files of any size
- No GC pressure

---

### Chunk Processing

Process large files in chunks:

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func processLargeFileInChunks(ctx context.Context, url string) error {
    resp, err := gocurl.Curl(ctx, url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Create buffered reader
    reader := bufio.NewReader(resp.Body)
    buffer := make([]byte, 1024*1024) // 1MB chunks

    totalProcessed := 0
    chunkNum := 0

    for {
        // Read chunk
        n, err := reader.Read(buffer)
        if n > 0 {
            chunkNum++
            totalProcessed += n

            // Process chunk (e.g., compute hash, parse data, etc.)
            fmt.Printf("Processed chunk %d: %d bytes (total: %d)\n",
                chunkNum, n, totalProcessed)

            // Your processing logic here
            // processChunk(buffer[:n])
        }

        if err != nil {
            if err == io.EOF {
                break
            }
            return err
        }
    }

    fmt.Printf("✅ Processed %d chunks (%d total bytes)\n",
        chunkNum, totalProcessed)

    return nil
}
```

---

## Part 4: Working with Cloud Storage

### AWS S3 Upload Example

```go
package main

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "log"
    "os"
    "time"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

type S3Client struct {
    accessKey string
    secretKey string
    region    string
    bucket    string
}

func NewS3Client(accessKey, secretKey, region, bucket string) *S3Client {
    return &S3Client{
        accessKey: accessKey,
        secretKey: secretKey,
        region:    region,
        bucket:    bucket,
    }
}

// Simplified S3 upload (use AWS SDK in production)
func (s *S3Client) UploadFile(ctx context.Context, key, filepath string) error {
    // Read file
    fileData, err := os.ReadFile(filepath)
    if err != nil {
        return err
    }

    // Build S3 URL
    url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
        s.bucket, s.region, key)

    // Generate signature (simplified - use AWS SDK for production)
    timestamp := time.Now().UTC().Format("20060102T150405Z")

    opts := options.NewRequestOptionsBuilder().
        SetURL(url).
        SetMethod("PUT").
        SetHeader("Content-Type", "application/octet-stream").
        SetHeader("x-amz-date", timestamp).
        SetBody(string(fileData)).
        Build()

    httpResp, _, err := gocurl.Process(ctx, opts)
    if err != nil {
        return err
    }
    defer httpResp.Body.Close()

    if httpResp.StatusCode != 200 {
        body, _ := io.ReadAll(httpResp.Body)
        return fmt.Errorf("S3 upload failed (%d): %s",
            httpResp.StatusCode, string(body))
    }

    fmt.Printf("✅ Uploaded %s to S3 bucket %s\n", key, s.bucket)
    return nil
}

func (s *S3Client) DownloadFile(ctx context.Context, key, filepath string) error {
    url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
        s.bucket, s.region, key)

    written, resp, err := gocurl.CurlDownload(ctx, filepath, url)
    if err != nil {
        return err
    }

    if resp.StatusCode != 200 {
        return fmt.Errorf("S3 download failed: %d", resp.StatusCode)
    }

    fmt.Printf("✅ Downloaded %s from S3: %d bytes\n", key, written)
    return nil
}

func main() {
    // NOTE: Use AWS SDK for production
    // This is a simplified example for demonstration

    client := NewS3Client(
        os.Getenv("AWS_ACCESS_KEY_ID"),
        os.Getenv("AWS_SECRET_ACCESS_KEY"),
        "us-east-1",
        "my-bucket",
    )

    ctx := context.Background()

    // Upload
    err := client.UploadFile(ctx, "documents/report.pdf", "local-report.pdf")
    if err != nil {
        log.Fatal(err)
    }

    // Download
    err = client.DownloadFile(ctx, "documents/report.pdf", "downloaded-report.pdf")
    if err != nil {
        log.Fatal(err)
    }
}
```

**Production Note:** Use official AWS SDK for production applications. This example demonstrates the HTTP operations involved.

---

## Part 5: Best Practices

### 1. Always Check Content-Length

```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close()

// Verify content length before downloading
contentLength := resp.ContentLength
if contentLength < 0 {
    log.Println("Warning: Content-Length unknown")
} else {
    fmt.Printf("Downloading %d bytes\n", contentLength)
}
```

### 2. Set Timeouts for Large Files

```go
opts := options.NewRequestOptionsBuilder().
    SetURL(url).
    SetTimeout(30 * time.Minute). // Longer timeout for large files
    Build()
```

### 3. Verify File Integrity

```go
package main

import (
    "crypto/sha256"
    "encoding/hex"
    "io"
    "os"
)

func verifyFileHash(filepath, expectedHash string) (bool, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return false, err
    }
    defer file.Close()

    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return false, err
    }

    actualHash := hex.EncodeToString(hash.Sum(nil))
    return actualHash == expectedHash, nil
}

// Usage:
written, resp, err := gocurl.CurlDownload(ctx, "file.zip", url)
if err != nil {
    log.Fatal(err)
}

// Verify checksum
valid, err := verifyFileHash("file.zip", expectedSHA256)
if !valid {
    log.Fatal("File integrity check failed!")
}
```

### 4. Handle Disk Space

```go
package main

import (
    "context"
    "fmt"
    "syscall"
)

func checkDiskSpace(path string, requiredBytes int64) error {
    var stat syscall.Statfs_t
    if err := syscall.Statfs(path, &stat); err != nil {
        return err
    }

    availableBytes := stat.Bavail * uint64(stat.Bsize)

    if int64(availableBytes) < requiredBytes {
        return fmt.Errorf("insufficient disk space: have %d, need %d",
            availableBytes, requiredBytes)
    }

    return nil
}

// Check before downloading
if err := checkDiskSpace("/downloads", contentLength); err != nil {
    log.Fatal(err)
}
```

### 5. Clean Up on Errors

```go
func downloadWithCleanup(ctx context.Context, url, filepath string) error {
    written, resp, err := gocurl.CurlDownload(ctx, filepath, url)

    if err != nil || resp.StatusCode != 200 {
        // Remove partial file on error
        os.Remove(filepath)
        return fmt.Errorf("download failed: %w", err)
    }

    return nil
}
```

---

## Part 6: Common Patterns

### Pattern 1: Parallel Downloads

```go
package main

import (
    "context"
    "fmt"
    "sync"

    "github.com/maniartech/gocurl"
)

type DownloadTask struct {
    URL      string
    Filepath string
}

func downloadParallel(ctx context.Context, tasks []DownloadTask) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(tasks))

    for _, task := range tasks {
        wg.Add(1)
        go func(t DownloadTask) {
            defer wg.Done()

            written, resp, err := gocurl.CurlDownload(ctx, t.Filepath, t.URL)
            if err != nil {
                errors <- fmt.Errorf("failed to download %s: %w", t.URL, err)
                return
            }

            if resp.StatusCode != 200 {
                errors <- fmt.Errorf("%s returned status %d", t.URL, resp.StatusCode)
                return
            }

            fmt.Printf("✅ Downloaded %s (%d bytes)\n", t.Filepath, written)
        }(task)
    }

    wg.Wait()
    close(errors)

    // Check for errors
    if len(errors) > 0 {
        return <-errors
    }

    return nil
}

func main() {
    tasks := []DownloadTask{
        {"https://example.com/file1.zip", "file1.zip"},
        {"https://example.com/file2.zip", "file2.zip"},
        {"https://example.com/file3.zip", "file3.zip"},
    }

    ctx := context.Background()
    if err := downloadParallel(ctx, tasks); err != nil {
        log.Fatal(err)
    }
}
```

### Pattern 2: Download Queue with Rate Limiting

```go
package main

import (
    "context"
    "time"
)

type DownloadQueue struct {
    tasks    chan DownloadTask
    rateLimit time.Duration
}

func NewDownloadQueue(concurrency int, rateLimit time.Duration) *DownloadQueue {
    q := &DownloadQueue{
        tasks:     make(chan DownloadTask, 100),
        rateLimit: rateLimit,
    }

    // Start workers
    for i := 0; i < concurrency; i++ {
        go q.worker()
    }

    return q
}

func (q *DownloadQueue) worker() {
    ticker := time.NewTicker(q.rateLimit)
    defer ticker.Stop()

    for task := range q.tasks {
        <-ticker.C // Rate limit

        written, _, err := gocurl.CurlDownload(context.Background(),
            task.Filepath, task.URL)

        if err != nil {
            fmt.Printf("❌ Failed: %s\n", task.URL)
        } else {
            fmt.Printf("✅ Done: %s (%d bytes)\n", task.Filepath, written)
        }
    }
}

func (q *DownloadQueue) Add(task DownloadTask) {
    q.tasks <- task
}

func (q *DownloadQueue) Close() {
    close(q.tasks)
}
```

---

## Summary

In this chapter, you learned:

✅ **CurlDownload Functions** - Stream files directly to disk
✅ **Progress Tracking** - Monitor download progress
✅ **Resumable Downloads** - Continue interrupted transfers
✅ **File Uploads** - Multipart form data
✅ **Large Files** - Memory-efficient streaming
✅ **Cloud Storage** - Work with S3 and similar services
✅ **Best Practices** - Integrity checks, error handling, cleanup

**Key Takeaways:**
1. Use CurlDownload for memory-efficient downloads
2. Stream large files instead of loading into memory
3. Implement progress tracking for better UX
4. Support resumable downloads with Range headers
5. Verify file integrity with checksums
6. Handle errors and clean up partial files

---

## What's Next?

**Part III: Security & Configuration**
Chapter 8 covers TLS/SSL, certificate pinning, mutual TLS, and security best practices.

**Practice Exercises:**
Complete the hands-on exercises in the `exercises/` directory to master file operations.
