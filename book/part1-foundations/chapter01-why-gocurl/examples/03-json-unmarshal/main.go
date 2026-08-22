// Package main demonstrates automatic JSON unmarshaling using GoCurl.
// This example fetches GitHub user data and unmarshals it into a struct.
//
// Run: go run json_unmarshal.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

// User represents a GitHub user
type User struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
}

func main() {
	ctx := context.Background()

	var user User
	resp, err := gocurl.CurlJSON(ctx, &user,
		"https://api.github.com/users/octocat")

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("User: %s (%s)\n", user.Name, user.Login)
	fmt.Printf("Repos: %d | Followers: %d\n",
		user.PublicRepos, user.Followers)
}
