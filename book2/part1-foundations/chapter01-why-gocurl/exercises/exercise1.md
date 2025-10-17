# Exercise 1: Build a Weather API Client

**Difficulty:** ⭐ Beginner
**Time:** 20-30 minutes
**Concepts:** GET requests, JSON parsing, error handling

## Objective

Create a simple weather API client that fetches current weather data for a given city using the OpenWeatherMap API (or any free weather API).

## Requirements

### Functional Requirements

1. Accept a city name as input (e.g., "London", "New York")
2. Make a GET request to a weather API
3. Parse the JSON response
4. Display the following information:
   - City name
   - Current temperature
   - Weather description (e.g., "Clear sky", "Light rain")
   - Humidity percentage
   - Wind speed

### Technical Requirements

1. Use `gocurl.CurlJSON()` for automatic JSON unmarshaling
2. Implement proper error handling
3. Create a struct to represent the weather data
4. Format the output nicely for users

## Getting Started

### 1. Choose a Weather API

**Option A: OpenWeatherMap** (Recommended)
- Free tier: https://openweathermap.org/api
- Sign up and get API key
- API endpoint: `https://api.openweathermap.org/data/2.5/weather?q={city}&appid={API_KEY}&units=metric`

**Option B: WeatherAPI.com**
- Free tier: https://www.weatherapi.com/
- API endpoint: `http://api.weatherapi.com/v1/current.json?key={API_KEY}&q={city}`

### 2. Create Project Structure

```bash
mkdir exercise1
cd exercise1
touch main.go
go mod init weather-client
go get github.com/maniartech/gocurl
```

### 3. Define Your Structs

Think about what data you need from the API response. Example structure:

```go
type WeatherResponse struct {
    // Add fields based on API response
    // Hint: Use json.org to format sample JSON and generate structs
}
```

### 4. Implementation Checklist

- [ ] Create context with timeout (5 seconds)
- [ ] Build API URL with city name and API key
- [ ] Make request using `gocurl.CurlJSON()`
- [ ] Handle errors (network, API errors, parsing)
- [ ] Extract and display weather information
- [ ] Format output for readability

## Example Output

```
Weather for London:
  Temperature: 15°C
  Conditions: Partly cloudy
  Humidity: 72%
  Wind Speed: 12 km/h
```

## Bonus Challenges

Once you've completed the basic requirements:

1. **Command-line arguments**: Accept city name as command-line argument instead of hardcoding
2. **Multiple cities**: Fetch weather for multiple cities in parallel
3. **Temperature units**: Support both Celsius and Fahrenheit
4. **Caching**: Cache results for 5 minutes to avoid repeated API calls
5. **Error messages**: Provide user-friendly error messages for common issues

## Hints

<details>
<summary>Hint 1: JSON Struct</summary>

Most weather APIs have nested JSON. You'll need nested structs:

```go
type WeatherResponse struct {
    Main struct {
        Temp     float64 `json:"temp"`
        Humidity int     `json:"humidity"`
    } `json:"main"`
    Weather []struct {
        Description string `json:"description"`
    } `json:"weather"`
    // ... more fields
}
```
</details>

<details>
<summary>Hint 2: Error Handling</summary>

Check both the error AND the HTTP status code:

```go
resp, err := gocurl.CurlJSON(ctx, &weather, url)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
    return fmt.Errorf("API error: status %d", resp.StatusCode)
}
```
</details>

<details>
<summary>Hint 3: Environment Variables</summary>

Store your API key securely:

```go
apiKey := os.Getenv("WEATHER_API_KEY")
if apiKey == "" {
    log.Fatal("WEATHER_API_KEY environment variable not set")
}
```
</details>

## Testing Your Solution

Test with various inputs:
- Valid city names: "London", "Tokyo", "New York"
- Invalid city names: "NonexistentCity"
- Empty input
- Cities with spaces: "Los Angeles"

## Next Steps

After completing this exercise:
1. Compare your solution with the provided solution
2. Review how error handling could be improved
3. Think about how you'd extend this for a production application
4. Move on to Exercise 2 for a more challenging task!

## Solution

The complete solution is available in `solutions/exercise1/main.go`. Try to complete the exercise before looking at it!
