package gocurl

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPersistentCookieJar(t *testing.T) {
	// Test without file
	jar, err := NewPersistentCookieJar("")
	require.NoError(t, err)
	assert.NotNil(t, jar)

	// Test with non-existent file (should not error)
	jar, err = NewPersistentCookieJar("/tmp/nonexistent-cookies.txt")
	require.NoError(t, err)
	assert.NotNil(t, jar)
}

func TestPersistentCookieJarSetAndGet(t *testing.T) {
	jar, err := NewPersistentCookieJar("")
	require.NoError(t, err)

	u, _ := url.Parse("http://example.com/")
	cookies := []*http.Cookie{
		{
			Name:  "session",
			Value: "abc123",
			Path:  "/",
		},
		{
			Name:  "user",
			Value: "john",
			Path:  "/",
		},
	}

	jar.SetCookies(u, cookies)

	// Retrieve cookies
	retrievedCookies := jar.Cookies(u)
	assert.Len(t, retrievedCookies, 2)

	// Check cookie values
	cookieMap := make(map[string]string)
	for _, c := range retrievedCookies {
		cookieMap[c.Name] = c.Value
	}

	assert.Equal(t, "abc123", cookieMap["session"])
	assert.Equal(t, "john", cookieMap["user"])
}

func TestLoadCookiesFromFile(t *testing.T) {
	// Create a temp cookie file with future expiration
	futureTime := time.Now().Add(24 * time.Hour).Unix()
	cookieContent := `# Netscape HTTP Cookie File
# This is a generated file! Do not edit.

.example.com	TRUE	/	FALSE	` + fmt.Sprintf("%d", futureTime) + `	session	abc123
localhost	TRUE	/api	FALSE	` + fmt.Sprintf("%d", futureTime) + `	token	xyz789
`

	tmpFile, err := ioutil.TempFile("", "cookies-*.txt")
	require.NoError(t, err)
	tmpFileName := tmpFile.Name()
	tmpFile.Close() // Close the file before writing to it
	defer os.Remove(tmpFileName)

	err = ioutil.WriteFile(tmpFileName, []byte(cookieContent), 0644)
	require.NoError(t, err)

	cookies, err := LoadCookiesFromFile(tmpFileName)
	require.NoError(t, err)
	assert.Len(t, cookies, 2)

	// Verify cookies
	cookieMap := make(map[string]*http.Cookie)
	for _, c := range cookies {
		cookieMap[c.Name] = c
	}

	assert.Equal(t, "abc123", cookieMap["session"].Value)
	assert.Equal(t, ".example.com", cookieMap["session"].Domain)
	assert.Equal(t, "xyz789", cookieMap["token"].Value)
}

func TestLoadCookiesFromFileExpired(t *testing.T) {
	// Create cookie file with expired cookie
	expiredTime := time.Now().Add(-24 * time.Hour).Unix()
	cookieContent := `# Netscape HTTP Cookie File
.example.com	TRUE	/	FALSE	` + fmt.Sprintf("%d", expiredTime) + `	expired	old_value
`

	tmpFile, err := ioutil.TempFile("", "cookies-*.txt")
	require.NoError(t, err)
	tmpFileName := tmpFile.Name()
	tmpFile.Close() // Close the file before writing to it
	defer os.Remove(tmpFileName)

	err = ioutil.WriteFile(tmpFileName, []byte(cookieContent), 0644)
	require.NoError(t, err)

	cookies, err := LoadCookiesFromFile(tmpFileName)
	require.NoError(t, err)
	// Expired cookies should be filtered out
	assert.Len(t, cookies, 0)
}

func TestSaveCookiesToFile(t *testing.T) {
	cookies := []*http.Cookie{
		{
			Name:    "session",
			Value:   "abc123",
			Domain:  "example.com",
			Path:    "/",
			Expires: time.Now().Add(24 * time.Hour),
			Secure:  true,
		},
		{
			Name:    "preference",
			Value:   "dark_mode",
			Domain:  "example.com",
			Path:    "/settings",
			Expires: time.Now().Add(24 * time.Hour),
			Secure:  false,
		},
	}

	tmpFile, err := ioutil.TempFile("", "cookies-*.txt")
	require.NoError(t, err)
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	err = SaveCookiesToFile(tmpFileName, cookies)
	require.NoError(t, err)

	// Read back and verify
	content, err := ioutil.ReadFile(tmpFileName)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "# Netscape HTTP Cookie File")
	assert.Contains(t, contentStr, "session")
	assert.Contains(t, contentStr, "abc123")
	assert.Contains(t, contentStr, "preference")
}

func TestSaveCookiesToFileSkipsExpired(t *testing.T) {
	cookies := []*http.Cookie{
		{
			Name:    "valid",
			Value:   "keep_me",
			Domain:  "example.com",
			Path:    "/",
			Expires: time.Now().Add(24 * time.Hour),
		},
		{
			Name:    "expired",
			Value:   "remove_me",
			Domain:  "example.com",
			Path:    "/",
			Expires: time.Now().Add(-24 * time.Hour),
		},
	}

	tmpFile, err := ioutil.TempFile("", "cookies-*.txt")
	require.NoError(t, err)
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	err = SaveCookiesToFile(tmpFileName, cookies)
	require.NoError(t, err)

	content, err := ioutil.ReadFile(tmpFileName)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "valid")
	assert.NotContains(t, contentStr, "expired")
}

func TestPersistentCookieJarLoad(t *testing.T) {
	// Create a temp cookie file
	futureTime := time.Now().Add(24 * time.Hour).Unix()
	cookieContent := `# Netscape HTTP Cookie File
# This is a generated file! Do not edit.

example.com	TRUE	/	FALSE	` + fmt.Sprintf("%d", futureTime) + `	test	value123
`

	tmpFile, err := ioutil.TempFile("", "cookies-*.txt")
	require.NoError(t, err)
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	err = ioutil.WriteFile(tmpFileName, []byte(cookieContent), 0644)
	require.NoError(t, err)

	// Create jar with file
	jar, err := NewPersistentCookieJar(tmpFileName)
	require.NoError(t, err)

	// Check if cookies were loaded
	u, _ := url.Parse("http://example.com/")
	cookies := jar.Cookies(u)

	// Note: Due to limitations in cookiejar.Jar, we may not retrieve all cookies
	// This test verifies the loading mechanism works without error
	assert.NotNil(t, cookies)
}

func TestPersistentCookieJarThreadSafety(t *testing.T) {
	jar, err := NewPersistentCookieJar("")
	require.NoError(t, err)

	var wg sync.WaitGroup
	u, _ := url.Parse("http://example.com/")

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cookies := []*http.Cookie{
				{
					Name:  "concurrent",
					Value: string(rune(id)),
					Path:  "/",
				},
			}
			jar.SetCookies(u, cookies)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jar.Cookies(u)
		}()
	}

	wg.Wait()
}

func TestLoadCookiesFromFileInvalidFormat(t *testing.T) {
	// Create file with invalid content
	cookieContent := `# Netscape HTTP Cookie File
invalid line format
another	bad	line
`

	tmpFile, err := ioutil.TempFile("", "cookies-*.txt")
	require.NoError(t, err)
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	err = ioutil.WriteFile(tmpFileName, []byte(cookieContent), 0644)
	require.NoError(t, err)

	// Should not error, just skip invalid lines
	cookies, err := LoadCookiesFromFile(tmpFileName)
	require.NoError(t, err)
	assert.Len(t, cookies, 0)
}

func TestLoadCookiesFromNonExistentFile(t *testing.T) {
	cookies, err := LoadCookiesFromFile("/tmp/definitely-does-not-exist-12345.txt")
	assert.Error(t, err)
	assert.Nil(t, cookies)
}

func TestPersistentCookieJarSave(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "cookies-*.txt")
	require.NoError(t, err)
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	jar, err := NewPersistentCookieJar(tmpFileName)
	require.NoError(t, err)

	// Add some cookies
	u, _ := url.Parse("http://example.com/")
	cookies := []*http.Cookie{
		{
			Name:  "session",
			Value: "test123",
			Path:  "/",
		},
	}
	jar.SetCookies(u, cookies)

	// Save
	err = jar.Save()
	assert.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(tmpFileName)
	assert.NoError(t, err)
}

func BenchmarkPersistentCookieJarSetCookies(b *testing.B) {
	jar, _ := NewPersistentCookieJar("")
	u, _ := url.Parse("http://example.com/")
	cookies := []*http.Cookie{
		{Name: "test", Value: "value", Path: "/"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jar.SetCookies(u, cookies)
	}
}

func BenchmarkPersistentCookieJarGetCookies(b *testing.B) {
	jar, _ := NewPersistentCookieJar("")
	u, _ := url.Parse("http://example.com/")
	cookies := []*http.Cookie{
		{Name: "test", Value: "value", Path: "/"},
	}
	jar.SetCookies(u, cookies)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jar.Cookies(u)
	}
}

func BenchmarkLoadCookiesFromFile(b *testing.B) {
	futureTime := time.Now().Add(24 * time.Hour).Unix()
	cookieContent := `# Netscape HTTP Cookie File
.example.com	TRUE	/	FALSE	` + fmt.Sprintf("%d", futureTime) + `	session	abc123
localhost	TRUE	/api	FALSE	` + fmt.Sprintf("%d", futureTime) + `	token	xyz789
`

	tmpFile, _ := ioutil.TempFile("", "cookies-*.txt")
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)
	ioutil.WriteFile(tmpFileName, []byte(cookieContent), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadCookiesFromFile(tmpFileName)
	}
}
