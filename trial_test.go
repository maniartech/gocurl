package gocurl_test

import (
	"context"
	"testing"

	"github.com/maniartech/gocurl"
)

func TestTrial(t *testing.T) {
	res, _, err := gocurl.Process(context.Background(), &gocurl.RequestOptions{
		Method: "GET",
		URL:    "https://example.com",
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	})

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	t.Logf("Response: %v", res)
	// t.Logf("Body: %v", body)
}
