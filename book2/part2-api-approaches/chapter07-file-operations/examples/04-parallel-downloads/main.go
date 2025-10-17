package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maniartech/gocurl"
)

// DownloadTask represents a single download task
type DownloadTask struct {
	URL      string
	Filepath string
	Name     string
}

// DownloadResult contains the result of a download
type DownloadResult struct {
	Task     DownloadTask
	Written  int64
	Duration time.Duration
	Error    error
}

func downloadFile(ctx context.Context, task DownloadTask) DownloadResult {
	start := time.Now()

	written, resp, err := gocurl.CurlDownload(ctx, task.Filepath, task.URL)

	result := DownloadResult{
		Task:     task,
		Written:  written,
		Duration: time.Since(start),
		Error:    err,
	}

	if err != nil {
		return result
	}

	if resp.StatusCode != 200 {
		result.Error = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return result
}

func downloadParallel(ctx context.Context, tasks []DownloadTask, concurrency int) []DownloadResult {
	results := make([]DownloadResult, len(tasks))
	taskChan := make(chan int, len(tasks))
	var wg sync.WaitGroup

	// Fill task channel
	for i := range tasks {
		taskChan <- i
	}
	close(taskChan)

	// Start workers
	fmt.Printf("Starting %d workers for %d downloads...\n\n", concurrency, len(tasks))

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for taskIdx := range taskChan {
				task := tasks[taskIdx]
				fmt.Printf("[Worker %d] Downloading %s...\n", workerID, task.Name)

				result := downloadFile(ctx, task)
				results[taskIdx] = result

				if result.Error != nil {
					fmt.Printf("[Worker %d] ❌ %s failed: %v\n",
						workerID, task.Name, result.Error)
				} else {
					fmt.Printf("[Worker %d] ✅ %s complete (%d bytes, %.2fs)\n",
						workerID, task.Name, result.Written, result.Duration.Seconds())
				}
			}
		}(w)
	}

	wg.Wait()
	return results
}

func main() {
	fmt.Println("Example 4: Parallel Downloads")
	fmt.Println("==============================\n")

	ctx := context.Background()

	// Define multiple download tasks
	tasks := []DownloadTask{
		{
			URL:      "https://httpbin.org/bytes/102400", // 100KB
			Filepath: "file1.bin",
			Name:     "File 1",
		},
		{
			URL:      "https://httpbin.org/bytes/204800", // 200KB
			Filepath: "file2.bin",
			Name:     "File 2",
		},
		{
			URL:      "https://httpbin.org/bytes/153600", // 150KB
			Filepath: "file3.bin",
			Name:     "File 3",
		},
		{
			URL:      "https://httpbin.org/bytes/256000", // 250KB
			Filepath: "file4.bin",
			Name:     "File 4",
		},
		{
			URL:      "https://httpbin.org/bytes/128000", // 125KB
			Filepath: "file5.bin",
			Name:     "File 5",
		},
	}

	// Download with 3 concurrent workers
	concurrency := 3
	startTime := time.Now()

	results := downloadParallel(ctx, tasks, concurrency)

	totalDuration := time.Since(startTime)

	// Print summary
	fmt.Println("\n" + string(make([]byte, 50)))
	fmt.Println("Download Summary")
	fmt.Println(string(make([]byte, 50)) + "\n")

	var totalBytes int64
	successCount := 0
	failCount := 0

	for i, result := range results {
		fmt.Printf("%d. %s\n", i+1, result.Task.Name)
		if result.Error != nil {
			fmt.Printf("   Status: ❌ Failed - %v\n", result.Error)
			failCount++
		} else {
			fmt.Printf("   Status: ✅ Success\n")
			fmt.Printf("   Size: %d bytes (%.2f KB)\n",
				result.Written, float64(result.Written)/1024)
			fmt.Printf("   Time: %.2fs\n", result.Duration.Seconds())
			totalBytes += result.Written
			successCount++
		}
		fmt.Println()
	}

	fmt.Printf("Total Downloads: %d\n", len(tasks))
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)
	fmt.Printf("Total Size: %d bytes (%.2f MB)\n",
		totalBytes, float64(totalBytes)/(1024*1024))
	fmt.Printf("Total Time: %.2fs\n", totalDuration.Seconds())

	if successCount > 0 {
		avgSpeed := float64(totalBytes) / totalDuration.Seconds() / 1024
		fmt.Printf("Average Speed: %.2f KB/s\n", avgSpeed)
	}

	fmt.Println("\nKey Features:")
	fmt.Println("  • Concurrent downloads with worker pool")
	fmt.Println("  • Configurable concurrency level")
	fmt.Println("  • Individual error handling per file")
	fmt.Println("  • Progress tracking per worker")
	fmt.Println("  • Summary statistics")
}
