# Exercise 2: Multi-API Data Aggregator

**Difficulty:** ⭐⭐ Intermediate
**Time:** 45-60 minutes
**Concepts:** Concurrent requests, multiple APIs, data aggregation, goroutines

## Objective

Build a developer profile aggregator that fetches information from multiple sources (GitHub, Stack Overflow, Twitter/X) and combines it into a unified profile view.

## Requirements

### Functional Requirements

1. Accept a username as input
2. Fetch data from multiple sources concurrently:
   - GitHub: User profile, repositories, followers
   - Stack Overflow: Reputation, badges, top tags (optional)
   - Any other developer profile API
3. Aggregate the data into a single profile
4. Handle API failures gracefully (show partial data if some APIs fail)
5. Display results in a formatted output

### Technical Requirements

1. Use goroutines to make concurrent requests
2. Use channels to collect results
3. Implement timeout for the entire operation (10 seconds)
4. Use `gocurl.CurlJSON()` for all API calls
5. Handle different response structures from different APIs
6. Provide informative error messages

## Getting Started

### 1. API Selection

**Required:**
- GitHub API: `https://api.github.com/users/{username}`
- GitHub Repos: `https://api.github.com/users/{username}/repos`

**Optional:**
- Stack Overflow API: `https://api.stackexchange.com/2.3/users?site=stackoverflow&filter=withstring&inname={username}`

### 2. Project Structure

```bash
mkdir exercise2
cd exercise2
touch main.go
go mod init profile-aggregator
go get github.com/maniartech/gocurl
```

### 3. Design Your Data Structures

```go
type DeveloperProfile struct {
    Username      string
    Name          string
    Bio           string
    GitHub        *GitHubData
    StackOverflow *StackOverflowData
    // Add more sources
}

type GitHubData struct {
    PublicRepos int
    Followers   int
    Following   int
    TopRepos    []Repository
    // Add more fields
}

// Define more structs...
```

### 4. Implementation Checklist

- [ ] Create result struct for each API
- [ ] Write function for each API call
- [ ] Implement concurrent fetching with goroutines
- [ ] Use channels to collect results
- [ ] Implement timeout using context
- [ ] Handle partial failures gracefully
- [ ] Aggregate data from all sources
- [ ] Format and display the profile

## Architecture Diagram

```
┌─────────────┐
│   User      │
│   Input     │
└──────┬──────┘
       │
       v
┌─────────────────────┐
│  Profile Aggregator │
└──────┬──────────────┘
       │
       ├──────────────────┬──────────────────┐
       │                  │                  │
       v                  v                  v
┌──────────┐      ┌──────────┐      ┌──────────┐
│  GitHub  │      │  Stack   │      │  Other   │
│   API    │      │ Overflow │      │   APIs   │
│ (goroutine)     │(goroutine)      │(goroutine)│
└────┬─────┘      └────┬─────┘      └────┬─────┘
     │                 │                  │
     └─────────────────┴──────────────────┘
                       │
                       v
                 ┌────────────┐
                 │  Channels  │
                 │  Collect   │
                 └─────┬──────┘
                       │
                       v
                 ┌────────────┐
                 │ Aggregate  │
                 │  Display   │
                 └────────────┘
```

## Example Output

```
Developer Profile: octocat
══════════════════════════════════════════

Name: The Octocat
Bio: GitHub mascot

GitHub Stats:
  Public Repos: 8
  Followers: 9,762
  Following: 9

  Top Repositories:
  1. Hello-World (Stars: 2,145)
  2. octocat.github.io (Stars: 892)
  3. Spoon-Knife (Stars: 456)

Stack Overflow Stats:
  Reputation: 1,234
  Gold Badges: 2
  Silver Badges: 15
  Bronze Badges: 45

Profile fetched in 2.3 seconds
```

## Code Structure Suggestion

```go
func main() {
    username := "octocat" // Or get from args
    profile, err := fetchDeveloperProfile(username)
    // Handle and display...
}

func fetchDeveloperProfile(username string) (*DeveloperProfile, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Create channels for results
    githubCh := make(chan *GitHubData)
    stackoverflowCh := make(chan *StackOverflowData)

    // Launch goroutines
    go fetchGitHubData(ctx, username, githubCh)
    go fetchStackOverflowData(ctx, username, stackoverflowCh)

    // Collect results
    // ...
}

func fetchGitHubData(ctx context.Context, username string, ch chan<- *GitHubData) {
    // Implement GitHub API calls
    // Send results to channel
}
```

## Bonus Challenges

1. **Rate Limiting**: Handle GitHub API rate limits gracefully
2. **Caching**: Cache API results for 1 hour
3. **Retry Logic**: Retry failed requests with exponential backoff
4. **More APIs**: Add LinkedIn, Medium, or Dev.to profiles
5. **Export**: Export profile to JSON or Markdown file
6. **Web Interface**: Create simple HTTP server to serve profiles
7. **Comparison**: Compare two developers side-by-side

## Hints

<details>
<summary>Hint 1: Concurrent Fetching Pattern</summary>

```go
type apiResult struct {
    data interface{}
    err  error
}

func fetchConcurrently(ctx context.Context, username string) (*Profile, error) {
    results := make(chan apiResult, 2)

    go func() {
        data, err := fetchAPI1(ctx, username)
        results <- apiResult{data, err}
    }()

    go func() {
        data, err := fetchAPI2(ctx, username)
        results <- apiResult{data, err}
    }()

    // Collect results with timeout
    for i := 0; i < 2; i++ {
        select {
        case result := <-results:
            // Handle result
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
}
```
</details>

<details>
<summary>Hint 2: Handling Partial Failures</summary>

Don't fail if one API is down. Collect what you can:

```go
profile := &DeveloperProfile{Username: username}

// Try each API, continue on errors
if githubData, err := fetchGitHub(ctx, username); err == nil {
    profile.GitHub = githubData
} else {
    log.Printf("GitHub API failed: %v", err)
}

// Even if GitHub failed, try Stack Overflow
if soData, err := fetchStackOverflow(ctx, username); err == nil {
    profile.StackOverflow = soData
}

return profile, nil // Return partial data
```
</details>

<details>
<summary>Hint 3: Context Timeout</summary>

Share the context with all goroutines:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Pass ctx to all API calls
go fetchAPI1(ctx, username, ch1)
go fetchAPI2(ctx, username, ch2)

// All calls will be cancelled after 10 seconds
```
</details>

## Testing Your Solution

Test scenarios:
1. **Valid username**: "octocat" (exists on all platforms)
2. **Partial presence**: User exists on GitHub but not Stack Overflow
3. **Invalid username**: "this-user-does-not-exist-12345"
4. **Slow API**: Verify timeout works (add artificial delay)
5. **Network error**: Disconnect internet mid-request

## Common Pitfalls

❌ **Mistake**: Not using channels properly
✅ **Solution**: Always close channels when done, use buffered channels for goroutines

❌ **Mistake**: Not handling context cancellation
✅ **Solution**: Pass context to all `gocurl` calls

❌ **Mistake**: Failing entirely if one API fails
✅ **Solution**: Handle errors gracefully, show partial data

❌ **Mistake**: Not setting timeouts
✅ **Solution**: Always use `context.WithTimeout`

## Next Steps

After completing this exercise:
1. Review concurrency patterns used
2. Think about how to scale to 10+ APIs
3. Consider how to make this production-ready
4. Move on to Exercise 3 for advanced patterns!

## Solution

The complete solution is available in `solutions/exercise2/main.go`.
