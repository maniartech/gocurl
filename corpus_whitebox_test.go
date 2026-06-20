package gocurl

import (
	"testing"

	"github.com/maniartech/gocurl/internal/corpus"
)

// TestCurlCompatCorpus_Parse is the headline regression guard: every real curl
// command from the docs corpus must parse to exactly its expected
// *options.RequestOptions (method, lossless URL, headers, verbatim body, auth).
// Adding a documented command is a one-line append to internal/corpus/compat.json.
func TestCurlCompatCorpus_Parse(t *testing.T) {
	cases := corpus.Load()
	if len(cases) < 9 {
		t.Fatalf("corpus has %d cases, want >= 9 (>=3 each for stripe/github/openai)", len(cases))
	}

	byVendor := map[string]int{}
	for _, c := range cases {
		byVendor[c.Vendor]++
		t.Run(c.Name, func(t *testing.T) {
			opts := parseCmd(t, c.Command)

			method := opts.Method
			if method == "" {
				method = "GET"
			}
			if method != c.Want.Method {
				t.Errorf("method = %q, want %q", method, c.Want.Method)
			}
			if opts.URL != c.Want.URL {
				t.Errorf("url = %q, want %q (must be lossless)", opts.URL, c.Want.URL)
			}
			if c.Want.Body != "" && opts.Body != c.Want.Body {
				t.Errorf("body = %q, want %q (verbatim)", opts.Body, c.Want.Body)
			}
			for k, v := range c.Want.Headers {
				if got := opts.Headers.Get(k); got != v {
					t.Errorf("header %s = %q, want %q", k, got, v)
				}
			}
			if c.Want.BasicUser != "" {
				if opts.BasicAuth == nil {
					t.Errorf("BasicAuth = nil, want user %q", c.Want.BasicUser)
				} else {
					if opts.BasicAuth.Username != c.Want.BasicUser {
						t.Errorf("basic user = %q, want %q", opts.BasicAuth.Username, c.Want.BasicUser)
					}
					if opts.BasicAuth.Password != c.Want.BasicPass {
						t.Errorf("basic pass = %q, want %q", opts.BasicAuth.Password, c.Want.BasicPass)
					}
				}
			}
		})
	}

	for _, v := range []string{"stripe", "github", "openai"} {
		if byVendor[v] < 3 {
			t.Errorf("vendor %q has %d cases, want >= 3", v, byVendor[v])
		}
	}
}
