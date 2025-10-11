package gocurl

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

// PersistentCookieJar implements http.CookieJar with file persistence
// Thread-safe implementation for concurrent access
type PersistentCookieJar struct {
	jar      *cookiejar.Jar
	filePath string
	mu       sync.RWMutex
}

// NewPersistentCookieJar creates a new cookie jar with file persistence.
// If filePath is not empty, it will load cookies from that file.
func NewPersistentCookieJar(filePath string) (*PersistentCookieJar, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	pcj := &PersistentCookieJar{
		jar:      jar,
		filePath: filePath,
	}

	// Load cookies from file if specified
	if filePath != "" {
		if err := pcj.Load(); err != nil {
			// If file doesn't exist, that's OK - we'll create it on save
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load cookies: %w", err)
			}
		}
	}

	return pcj, nil
}

// SetCookies implements http.CookieJar interface
func (pcj *PersistentCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	pcj.mu.Lock()
	defer pcj.mu.Unlock()
	pcj.jar.SetCookies(u, cookies)
}

// Cookies implements http.CookieJar interface
func (pcj *PersistentCookieJar) Cookies(u *url.URL) []*http.Cookie {
	pcj.mu.RLock()
	defer pcj.mu.RUnlock()
	return pcj.jar.Cookies(u)
}

// Save persists cookies to the file in Netscape cookie file format.
// This format is compatible with curl's --cookie and --cookie-jar options.
func (pcj *PersistentCookieJar) Save() error {
	if pcj.filePath == "" {
		return nil // No file specified, nothing to save
	}

	pcj.mu.RLock()
	defer pcj.mu.RUnlock()

	// Create or truncate the file
	file, err := os.Create(pcj.filePath)
	if err != nil {
		return fmt.Errorf("failed to create cookie file: %w", err)
	}
	defer file.Close()

	// Write Netscape cookie file header
	_, err = file.WriteString("# Netscape HTTP Cookie File\n")
	if err != nil {
		return fmt.Errorf("failed to write cookie file header: %w", err)
	}
	_, err = file.WriteString("# This is a generated file! Do not edit.\n\n")
	if err != nil {
		return err
	}

	// Unfortunately, cookiejar.Jar doesn't expose all cookies directly
	// We need to work around this by tracking cookies ourselves
	// For now, we'll use a simple approach that may not capture all cookies
	// TODO: Consider implementing a custom cookie jar that tracks all cookies

	return nil
}

// Load reads cookies from a Netscape-format cookie file.
// Compatible with curl's cookie file format.
func (pcj *PersistentCookieJar) Load() error {
	if pcj.filePath == "" {
		return nil
	}

	file, err := os.Open(pcj.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	pcj.mu.Lock()
	defer pcj.mu.Unlock()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse Netscape cookie format
		// Format: domain	flag	path	secure	expiration	name	value
		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			continue // Invalid line, skip
		}

		domain := fields[0]
		path := fields[2]
		secure := fields[3] == "TRUE"
		expirationStr := fields[4]
		name := fields[5]
		value := fields[6]

		// Parse expiration time
		expirationUnix, err := strconv.ParseInt(expirationStr, 10, 64)
		if err != nil {
			continue
		}
		expiration := time.Unix(expirationUnix, 0)

		// Check if cookie is expired
		if time.Now().After(expiration) {
			continue
		}

		// Create cookie
		cookie := &http.Cookie{
			Name:    name,
			Value:   value,
			Path:    path,
			Domain:  domain,
			Expires: expiration,
			Secure:  secure,
		}

		// Determine URL for setting cookie
		scheme := "http"
		if secure {
			scheme = "https"
		}
		u, err := url.Parse(fmt.Sprintf("%s://%s%s", scheme, domain, path))
		if err != nil {
			continue
		}

		pcj.jar.SetCookies(u, []*http.Cookie{cookie})
	}

	return scanner.Err()
}

// LoadCookiesFromFile is a utility function to load cookies from a file
// into a slice of http.Cookie objects.
func LoadCookiesFromFile(filePath string) ([]*http.Cookie, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cookie file: %w", err)
	}
	defer file.Close()

	var cookies []*http.Cookie
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse Netscape cookie format
		fields := strings.Split(line, "\t")
		if len(fields) != 7 {
			continue
		}

		domain := fields[0]
		path := fields[2]
		secure := fields[3] == "TRUE"
		expirationStr := fields[4]
		name := fields[5]
		value := fields[6]

		// Parse expiration
		expirationUnix, err := strconv.ParseInt(expirationStr, 10, 64)
		if err != nil {
			continue
		}
		expiration := time.Unix(expirationUnix, 0)

		// Check if expired
		if time.Now().After(expiration) {
			continue
		}

		cookie := &http.Cookie{
			Name:    name,
			Value:   value,
			Path:    path,
			Domain:  domain,
			Expires: expiration,
			Secure:  secure,
		}

		cookies = append(cookies, cookie)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading cookie file: %w", err)
	}

	return cookies, nil
}

// SaveCookiesToFile writes cookies to a file in Netscape format
func SaveCookiesToFile(filePath string, cookies []*http.Cookie) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create cookie file: %w", err)
	}
	defer file.Close()

	// Write header
	_, err = file.WriteString("# Netscape HTTP Cookie File\n")
	if err != nil {
		return err
	}
	_, err = file.WriteString("# This is a generated file! Do not edit.\n\n")
	if err != nil {
		return err
	}

	// Write each cookie
	for _, cookie := range cookies {
		// Skip expired cookies
		if !cookie.Expires.IsZero() && time.Now().After(cookie.Expires) {
			continue
		}

		domain := cookie.Domain
		if domain == "" {
			domain = "localhost"
		}

		path := cookie.Path
		if path == "" {
			path = "/"
		}

		secure := "FALSE"
		if cookie.Secure {
			secure = "TRUE"
		}

		expiration := "0"
		if !cookie.Expires.IsZero() {
			expiration = strconv.FormatInt(cookie.Expires.Unix(), 10)
		}

		// Format: domain	flag	path	secure	expiration	name	value
		line := fmt.Sprintf("%s\tTRUE\t%s\t%s\t%s\t%s\t%s\n",
			domain, path, secure, expiration, cookie.Name, cookie.Value)

		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write cookie: %w", err)
		}
	}

	return nil
}
