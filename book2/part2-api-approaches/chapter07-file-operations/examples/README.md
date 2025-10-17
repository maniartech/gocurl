# Chapter 7 Examples: File Operations

This directory contains working examples demonstrating file download and upload operations with GoCurl.

## Examples Overview

### Downloads
1. **[01-basic-download](01-basic-download/)** - Simple file download with CurlDownload
2. **[02-progress-tracking](02-progress-tracking/)** - Download with progress bar
3. **[03-resumable-download](03-resumable-download/)** - Resume interrupted downloads
4. **[04-parallel-downloads](04-parallel-downloads/)** - Download multiple files concurrently

### Uploads
5. **[05-file-upload](05-file-upload/)** - Upload files with multipart form
6. **[06-upload-progress](06-upload-progress/)** - Upload with progress tracking

### Advanced
7. **[07-large-files](07-large-files/)** - Efficient large file handling
8. **[08-integrity-check](08-integrity-check/)** - Verify downloads with checksums

## Running Examples

```bash
cd 01-basic-download
go run main.go
```

## Prerequisites

- Go 1.18+
- Internet connection
- Write permissions for downloads

## Key Learning Points

- ✅ Downloading files with CurlDownload
- ✅ Streaming to avoid memory issues
- ✅ Progress tracking patterns
- ✅ Resumable downloads with Range headers
- ✅ Multipart file uploads
- ✅ Parallel downloads
- ✅ File integrity verification
- ✅ Error handling and cleanup

## APIs Used

- **httpbin.org** - Testing uploads
- **Various CDNs** - Real file downloads

## Safety Notes

- Examples download real files (can be large)
- Check disk space before running
- Some examples may take time to complete
- Clean up downloaded files when done

## Next Steps

1. Try examples with your own files/URLs
2. Complete exercises in `../exercises/`
3. Proceed to Part III: Security & Configuration
