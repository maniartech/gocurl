package gocurl_test

import (
	"context"
	"testing"

	"github.com/maniartech/gocurl"
)

func TestTrial(t *testing.T) {
	res, _, _ := gocurl.Curl(context.Background(), "https://example.com")

	t.Logf("Response: %#v", res)
	// t.Logf("Body: %v", body)
}
