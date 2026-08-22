// Example 3: Context Patterns
// Demonstrates timeout, cancellation, and deadline management with context.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Print("⏱️  Context Patterns Demonstration\n\n")

	// Pattern 1: Request timeout
	pattern1_Timeout()

	// Pattern 2: Manual cancellation
	fmt.Println()
	pattern2_Cancellation()

	// Pattern 3: Deadline
	fmt.Println()
	pattern3_Deadline()

	// Pattern 4: Context propagation
	fmt.Println()
	pattern4_Propagation()
}

func pattern1_Timeout() {
	fmt.Println("1️⃣  Pattern: Request Timeout")

	// Scenario A: Normal request with reasonable timeout
	fmt.Println("\n   📍 Scenario A: Normal request (5s timeout)")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	body, resp, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/1")
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("   ❌ Error: %v", err)
	} else {
		defer resp.Body.Close()
		fmt.Printf("   ✅ Success in %v\n", elapsed)
		fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
		fmt.Printf("   📏 Response: %d bytes\n", len(body))
	}

	// Scenario B: Request that will timeout
	fmt.Println("\n   📍 Scenario B: Slow request (1s timeout, 5s delay)")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()

	start = time.Now()
	_, _, err = gocurl.CurlString(ctx2, "https://httpbin.org/delay/5")
	elapsed = time.Since(start)

	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			fmt.Printf("   ⏰ Timed out after %v (as expected)\n", elapsed)
			fmt.Printf("   💡 Error: %v\n", err)
		} else {
			fmt.Printf("   ❌ Different error: %v\n", err)
		}
	}

	fmt.Println("\n   💡 Use timeouts to prevent hanging requests")
}

func pattern2_Cancellation() {
	fmt.Println("2️⃣  Pattern: Manual Cancellation")

	ctx, cancel := context.WithCancel(context.Background())

	// Start request in goroutine
	done := make(chan error)
	go func() {
		fmt.Println("   🚀 Starting request...")
		_, _, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/10")
		done <- err
	}()

	// Cancel after 500ms
	time.Sleep(500 * time.Millisecond)
	fmt.Println("   🛑 Cancelling request...")
	cancel()

	// Wait for result
	err := <-done
	if err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Println("   ✅ Request cancelled successfully")
		} else {
			fmt.Printf("   ❌ Error: %v\n", err)
		}
	}

	fmt.Println("   💡 Use cancellation for user-initiated stops")
}

func pattern3_Deadline() {
	fmt.Println("3️⃣  Pattern: Absolute Deadline")

	// Must complete by specific time (5 seconds from now)
	deadline := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	fmt.Printf("   ⏰ Deadline: %s\n", deadline.Format("15:04:05"))
	fmt.Println("   🚀 Starting request...")

	start := time.Now()
	_, resp, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/2")
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("   ❌ Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Completed in %v (before deadline)\n", elapsed)
	fmt.Printf("   ⏱️  Time remaining: %v\n", time.Until(deadline))
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)

	fmt.Println("   💡 Use deadlines for SLA requirements")
}

func pattern4_Propagation() {
	fmt.Println("4️⃣  Pattern: Context Propagation")

	// Parent context with timeout
	parentCtx, parentCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer parentCancel()

	// Child context with its own timeout
	childCtx, childCancel := context.WithTimeout(parentCtx, 3*time.Second)
	defer childCancel()

	fmt.Println("   👨 Parent timeout: 10 seconds")
	fmt.Println("   👶 Child timeout: 3 seconds")
	fmt.Println("   🚀 Making request with child context...")

	start := time.Now()
	body, resp, err := gocurl.CurlString(childCtx, "https://httpbin.org/delay/1")
	elapsed := time.Since(start)

	if err != nil {
		if childCtx.Err() == context.DeadlineExceeded {
			fmt.Printf("   ⏰ Child timeout triggered after %v\n", elapsed)
		} else if parentCtx.Err() == context.DeadlineExceeded {
			fmt.Printf("   ⏰ Parent timeout triggered after %v\n", elapsed)
		} else {
			log.Printf("   ❌ Error: %v", err)
		}
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Success in %v\n", elapsed)
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   📏 Response: %d bytes\n", len(body))

	fmt.Println("\n   💡 Child context inherits parent's cancellation")
	fmt.Println("   💡 Whichever timeout is shorter takes effect")
}
