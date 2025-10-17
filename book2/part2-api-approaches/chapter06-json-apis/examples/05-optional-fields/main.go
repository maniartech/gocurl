package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

// Product represents a product with optional fields
type Product struct {
	ID          int                     `json:"id"`
	Name        string                  `json:"name"`
	Description *string                 `json:"description"` // Optional - may be null
	Price       float64                 `json:"price"`
	Discount    *float64                `json:"discount"` // Optional - may be null
	Category    string                  `json:"category"`
	Stock       *int                    `json:"stock"`    // Optional - may be null
	Tags        []string                `json:"tags"`     // Empty array vs null
	Metadata    *map[string]interface{} `json:"metadata"` // Optional object
}

// MockProductResponse simulates API response for demonstration
type MockProductResponse struct {
	ID          int                     `json:"id"`
	Name        string                  `json:"name"`
	Description *string                 `json:"description"`
	Price       float64                 `json:"price"`
	Discount    *float64                `json:"discount"`
	Category    string                  `json:"category"`
	Stock       *int                    `json:"stock"`
	Tags        []string                `json:"tags"`
	Metadata    *map[string]interface{} `json:"metadata"`
}

func displayProduct(product Product) {
	fmt.Printf("\nProduct: %s (ID: %d)\n", product.Name, product.ID)
	fmt.Printf("Category: %s\n", product.Category)
	fmt.Printf("Price: $%.2f\n", product.Price)

	// Handle optional Description
	if product.Description != nil {
		fmt.Printf("Description: %s\n", *product.Description)
	} else {
		fmt.Println("Description: Not available")
	}

	// Handle optional Discount
	if product.Discount != nil {
		discountPercent := *product.Discount * 100
		finalPrice := product.Price * (1 - *product.Discount)
		fmt.Printf("Discount: %.0f%% off\n", discountPercent)
		fmt.Printf("Final Price: $%.2f\n", finalPrice)
	} else {
		fmt.Println("Discount: None")
	}

	// Handle optional Stock
	if product.Stock != nil {
		fmt.Printf("Stock: %d units\n", *product.Stock)
	} else {
		fmt.Println("Stock: Unknown")
	}

	// Handle Tags array (can be empty but not null)
	if len(product.Tags) > 0 {
		fmt.Printf("Tags: %v\n", product.Tags)
	} else {
		fmt.Println("Tags: None")
	}

	// Handle optional Metadata
	if product.Metadata != nil {
		fmt.Printf("Metadata: %v\n", *product.Metadata)
	} else {
		fmt.Println("Metadata: None")
	}
}

func main() {
	fmt.Println("Example 5: Handling Optional Fields and Null Values")
	fmt.Println("====================================================\n")

	ctx := context.Background()

	// Example 1: Product with all fields present
	fmt.Println("--- Product 1: All fields present ---")

	desc1 := "High-quality wireless headphones with noise cancellation"
	discount1 := 0.20 // 20% off
	stock1 := 50
	metadata1 := map[string]interface{}{
		"brand":    "AudioTech",
		"warranty": "2 years",
	}

	product1 := Product{
		ID:          101,
		Name:        "Wireless Headphones",
		Description: &desc1,
		Price:       199.99,
		Discount:    &discount1,
		Category:    "Electronics",
		Stock:       &stock1,
		Tags:        []string{"electronics", "audio", "wireless"},
		Metadata:    &metadata1,
	}

	displayProduct(product1)

	// Example 2: Product with some optional fields missing
	fmt.Println("\n--- Product 2: Some optional fields missing ---")

	stock2 := 100

	product2 := Product{
		ID:          102,
		Name:        "Basic USB Cable",
		Description: nil, // No description
		Price:       9.99,
		Discount:    nil, // No discount
		Category:    "Accessories",
		Stock:       &stock2,
		Tags:        []string{"accessories", "cable"},
		Metadata:    nil, // No metadata
	}

	displayProduct(product2)

	// Example 3: Product with minimal fields
	fmt.Println("\n--- Product 3: Minimal fields (out of stock) ---")

	product3 := Product{
		ID:          103,
		Name:        "Rare Vintage Item",
		Description: nil,
		Price:       499.99,
		Discount:    nil,
		Category:    "Collectibles",
		Stock:       nil,        // Stock unknown
		Tags:        []string{}, // Empty tags
		Metadata:    nil,
	}

	displayProduct(product3)

	// Demonstrate fetching from a real API (JSONPlaceholder)
	fmt.Println("\n--- Fetching from JSONPlaceholder API ---")

	type Post struct {
		UserID int    `json:"userId"`
		ID     int    `json:"id"`
		Title  string `json:"title"`
		Body   string `json:"body"`
	}

	var post Post
	resp, err := gocurl.CurlJSON(ctx, &post,
		"https://jsonplaceholder.typicode.com/posts/1")

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	fmt.Printf("\nStatus: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("\nPost %d by User %d:\n", post.ID, post.UserID)
	fmt.Printf("Title: %s\n", post.Title)
	fmt.Printf("Body: %s\n", post.Body)

	fmt.Println("\n✅ Key Takeaways:")
	fmt.Println("   • Use pointers (*Type) for optional fields that may be null")
	fmt.Println("   • Always check if pointer is nil before dereferencing")
	fmt.Println("   • Slices can be empty ([]) without being null")
	fmt.Println("   • Map pointers (*map[string]interface{}) for optional objects")
}
