package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TestResult struct {
	Path     string
	Status   string // "PASS", "FAIL", "SKIP"
	Error    string
	Duration time.Duration
}

var (
	partFilter    = flag.Int("part", 0, "Test only specific part (1 or 2)")
	chapterFilter = flag.Int("chapter", 0, "Test only specific chapter number")
	runExamples   = flag.Bool("run", false, "Actually run examples (not just compile)")
	verbose       = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	fmt.Println("==================================================")
	fmt.Println("Testing All Book Examples")
	fmt.Println("==================================================\n")

	// Show configuration
	if *verbose {
		fmt.Printf("Configuration:\n")
		if *partFilter > 0 {
			fmt.Printf("  Part Filter: %d\n", *partFilter)
		}
		if *chapterFilter > 0 {
			fmt.Printf("  Chapter Filter: %d\n", *chapterFilter)
		}
		fmt.Printf("  Run Examples: %v\n", *runExamples)
		fmt.Println()
	}

	var results []TestResult
	totalExamples := 0
	passedExamples := 0
	failedExamples := 0
	skippedExamples := 0

	// Test Part 1
	if *partFilter == 0 || *partFilter == 1 {
		fmt.Println("Testing Part 1: Foundations")
		fmt.Println("-----------------------------------")
		part1Results := testPart("book2/part1-foundations", 1)
		results = append(results, part1Results...)
		fmt.Println()
	}

	// Test Part 2
	if *partFilter == 0 || *partFilter == 2 {
		fmt.Println("Testing Part 2: API Approaches")
		fmt.Println("-----------------------------------")
		part2Results := testPart("book2/part2-api-approaches", 2)
		results = append(results, part2Results...)
		fmt.Println()
	}

	// Count results
	for _, result := range results {
		totalExamples++
		switch result.Status {
		case "PASS":
			passedExamples++
		case "FAIL":
			failedExamples++
		case "SKIP":
			skippedExamples++
		}
	}

	// Print summary
	fmt.Println("\n==================================================")
	fmt.Println("Test Summary")
	fmt.Println("==================================================\n")
	fmt.Printf("Total Examples:   %d\n", totalExamples)
	fmt.Printf("Passed:           %d ✓\n", passedExamples)
	fmt.Printf("Failed:           %d ✗\n", failedExamples)
	fmt.Printf("Skipped:          %d ⊘\n", skippedExamples)
	fmt.Println()

	// Show failed examples
	if failedExamples > 0 {
		fmt.Println("Failed Examples:")
		for _, result := range results {
			if result.Status == "FAIL" {
				fmt.Printf("  ❌ %s\n", result.Path)
				if result.Error != "" {
					lines := strings.Split(result.Error, "\n")
					for i, line := range lines {
						if i < 5 && line != "" { // Show first 5 lines
							fmt.Printf("     %s\n", line)
						}
					}
				}
			}
		}
		fmt.Println()
	}

	// Show timing if verbose
	if *verbose && len(results) > 0 {
		var totalDuration time.Duration
		for _, r := range results {
			if r.Status == "PASS" {
				totalDuration += r.Duration
			}
		}
		fmt.Printf("Total Build Time: %v\n", totalDuration.Round(time.Millisecond))
	}

	// Calculate success rate
	if totalExamples > 0 {
		successRate := (passedExamples * 100) / totalExamples
		fmt.Printf("Success Rate: %d%%\n\n", successRate)
	}

	// Exit with appropriate code
	if failedExamples > 0 {
		fmt.Println("❌ Some examples failed to compile!")
		os.Exit(1)
	} else {
		fmt.Println("✅ All examples compiled successfully!")
		os.Exit(0)
	}
}

func testPart(partPath string, partNum int) []TestResult {
	var results []TestResult

	// Find all chapter directories
	chapters, err := filepath.Glob(filepath.Join(partPath, "chapter*"))
	if err != nil {
		return results
	}

	for _, chapterPath := range chapters {
		// Apply chapter filter
		if *chapterFilter > 0 {
			chapterNum := extractChapterNumber(chapterPath)
			if chapterNum != *chapterFilter {
				continue
			}
		}
		examplesPath := filepath.Join(chapterPath, "examples")

		// Check if examples directory exists
		if _, err := os.Stat(examplesPath); os.IsNotExist(err) {
			continue
		}

		// Find all example directories
		examples, err := os.ReadDir(examplesPath)
		if err != nil {
			continue
		}

		for _, example := range examples {
			if !example.IsDir() {
				continue
			}

			examplePath := filepath.Join(examplesPath, example.Name())
			mainGoPath := filepath.Join(examplePath, "main.go")

			// Check if main.go exists
			if _, err := os.Stat(mainGoPath); os.IsNotExist(err) {
				result := TestResult{
					Path:   getRelativePath(examplePath),
					Status: "SKIP",
				}
				results = append(results, result)
				fmt.Printf("[%d] %s ... SKIP (no main.go)\n", len(results), result.Path)
				continue
			}

			// Try to build or run
			var result TestResult
			if *runExamples {
				result = runExample(examplePath)
			} else {
				result = buildExample(examplePath)
			}
			results = append(results, result)

			status := "✓"
			if result.Status == "FAIL" {
				status = "✗"
			}

			output := fmt.Sprintf("[%d] %s ... %s %s", len(results), result.Path, result.Status, status)
			if *verbose && result.Duration > 0 {
				output += fmt.Sprintf(" (%v)", result.Duration.Round(time.Millisecond))
			}
			fmt.Println(output)
		}
	}

	return results
}

func buildExample(examplePath string) TestResult {
	result := TestResult{
		Path:   getRelativePath(examplePath),
		Status: "PASS",
	}

	// Run go build
	start := time.Now()
	cmd := exec.Command("go", "build", "-o", os.DevNull, ".")
	cmd.Dir = examplePath
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = "FAIL"
		result.Error = string(output)
	}

	return result
}

func runExample(examplePath string) TestResult {
	result := TestResult{
		Path:   getRelativePath(examplePath),
		Status: "PASS",
	}

	// Run go run with timeout
	start := time.Now()
	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = examplePath

	// Set timeout
	done := make(chan error, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		if err != nil {
			result.Error = string(output)
			done <- err
		} else {
			done <- nil
		}
	}()

	// Wait with timeout
	select {
	case err := <-done:
		result.Duration = time.Since(start)
		if err != nil {
			result.Status = "FAIL"
		}
	case <-time.After(10 * time.Second):
		result.Duration = 10 * time.Second
		result.Status = "FAIL"
		result.Error = "timeout after 10 seconds"
		cmd.Process.Kill()
	}

	return result
}

func getRelativePath(path string) string {
	// Convert to relative path for display
	parts := strings.Split(filepath.ToSlash(path), "/")

	// Find "book2" and take everything after
	for i, part := range parts {
		if part == "book2" && i+1 < len(parts) {
			return strings.Join(parts[i+1:], "/")
		}
	}

	return path
}

func extractChapterNumber(chapterPath string) int {
	base := filepath.Base(chapterPath)
	// Extract number from "chapterNN-name"
	parts := strings.Split(base, "-")
	if len(parts) > 0 {
		numStr := strings.TrimPrefix(parts[0], "chapter")
		var num int
		fmt.Sscanf(numStr, "%d", &num)
		return num
	}
	return 0
}
