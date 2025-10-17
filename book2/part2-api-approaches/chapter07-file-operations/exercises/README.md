# Chapter 7 Exercises: File Operations

Complete these exercises to master file download and upload operations.

## Exercise Overview

| Exercise | Title | Difficulty | Duration |
|----------|-------|-----------|----------|
| 1 | Download & Upload Basics | Beginner | 30-40 min |
| 2 | Advanced File Operations | Intermediate | 45-60 min |
| 3 | Production File Manager | Advanced | 90-120 min |

## Prerequisites

- Completed Chapter 7 content
- Reviewed examples 01-04
- Go 1.18+ installed
- Sufficient disk space

## Exercises

### Exercise 1: Download & Upload Basics
**Focus:** CurlDownload, basic uploads, error handling

**Topics:**
- Download files with CurlDownload
- Check status codes
- Upload with multipart form
- File verification
- Basic progress tracking

**Skills:**
- Using CurlDownload functions
- Multipart form creation
- Error handling
- File I/O operations

---

### Exercise 2: Advanced File Operations
**Focus:** Progress tracking, resumable downloads, parallel operations

**Topics:**
- Progress tracking implementation
- Resumable downloads with Range headers
- Parallel downloads
- File integrity checks (checksums)
- Large file streaming

**Skills:**
- Progress reader implementation
- HTTP Range requests
- Concurrent file operations
- SHA256 checksums
- Memory-efficient patterns

---

### Exercise 3: Production File Manager
**Focus:** Complete file management system

**Topics:**
- Download queue with rate limiting
- Retry failed downloads
- Upload manager with progress
- File integrity verification
- Cloud storage integration
- Testing file operations

**Skills:**
- Production-ready architecture
- Queue management
- Rate limiting
- Comprehensive error handling
- Integration testing

## Getting Started

1. Read exercise requirements
2. Review related examples
3. Implement solution
4. Run validation tests
5. Check self-assessment criteria

## Learning Path

```
Exercise 1 (Basics)
    ↓
Review examples 01-04
    ↓
Exercise 2 (Advanced)
    ↓
Review examples 05-08
    ↓
Exercise 3 (Production)
```

## Key Skills Practiced

- ✅ File downloads with CurlDownload
- ✅ File uploads with multipart forms
- ✅ Progress tracking patterns
- ✅ Resumable download implementation
- ✅ Parallel file operations
- ✅ File integrity verification
- ✅ Error handling and recovery
- ✅ Memory-efficient streaming

## Additional Resources

- [Chapter 7 Content](../chapter.md)
- [Examples](../examples/)
- [Go io Package](https://pkg.go.dev/io)
- [HTTP Range Requests](https://developer.mozilla.org/en-US/docs/Web/HTTP/Range_requests)

## Next Steps

After completing all exercises:
1. Build a file sync utility
2. Create a backup system
3. Proceed to Part III: Security & Configuration
