package tests

import (
	"context"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/internal/corpus"
)

// TestCurlCompatCorpus_Execute runs every corpus command end-to-end against an
// httptest echo server (with the doc host swapped for the server) and asserts the
// parsed request reaches the wire intact — method, lossless path/query, verbatim
// body, and auth form. It complements the whitebox parse assertion in package
// gocurl: same manifest, both layers.
func TestCurlCompatCorpus_Execute(t *testing.T) {
	// Corpus commands carry -u basic auth against what becomes an http:// test
	// server, so opt out of the fail-closed plaintext-auth policy for this check.
	t.Setenv("GOCURL_ALLOW_INSECURE_AUTH", "1")

	for _, c := range corpus.Load() {
		t.Run(c.Name, func(t *testing.T) {
			es := newEchoServer(t)

			want, err := url.Parse(c.Want.URL)
			if err != nil {
				t.Fatalf("bad corpus url %q: %v", c.Want.URL, err)
			}
			// Point the doc command at the echo server by swapping scheme+host.
			schemeHost := want.Scheme + "://" + want.Host
			cmd := strings.Replace(c.Command, schemeHost, es.URL, 1)

			resp, err := gocurl.Curl(context.Background(), cmd)
			if err != nil {
				t.Fatalf("execute %q: %v", cmd, err)
			}
			resp.Body.Close()

			wantMethod := c.Want.Method
			if wantMethod == "" {
				wantMethod = "GET"
			}
			if es.lastMethod != wantMethod {
				t.Errorf("method = %q, want %q", es.lastMethod, wantMethod)
			}
			if es.lastPath != want.Path {
				t.Errorf("path = %q, want %q (lossless)", es.lastPath, want.Path)
			}
			if es.lastQuery != want.RawQuery {
				t.Errorf("query = %q, want %q", es.lastQuery, want.RawQuery)
			}
			if c.Want.Body != "" && es.lastBody != c.Want.Body {
				t.Errorf("wire body = %q, want %q (verbatim)", es.lastBody, c.Want.Body)
			}
			for k, v := range c.Want.Headers {
				if got := es.lastHeader.Get(k); got != v {
					t.Errorf("wire header %s = %q, want %q", k, got, v)
				}
			}
			if c.Want.BasicUser != "" {
				got := es.lastHeader.Get("Authorization")
				const p = "Basic "
				if !strings.HasPrefix(got, p) {
					t.Fatalf("Authorization = %q, want Basic auth", got)
				}
				dec, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(got, p))
				if string(dec) != c.Want.BasicUser+":"+c.Want.BasicPass {
					t.Errorf("basic auth = %q, want %q", dec, c.Want.BasicUser+":"+c.Want.BasicPass)
				}
			}
		})
	}
}
