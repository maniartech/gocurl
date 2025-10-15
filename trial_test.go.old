package gocurl_test

import (
	"context"
	"testing"

	"github.com/maniartech/gocurl"
)

func TestTrial(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping trial test with actual network request in short mode")
	}

	res, _, _ := gocurl.Curl(context.Background(), "https://example.com")
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}

	t.Logf("Response: %#v", res)
	// t.Logf("Body: %v", body)
}
