// Package corpus holds the curl-compat regression corpus: real command strings
// copied verbatim from public API docs, each mapped to the parse result it must
// produce. It is the regression guard for gocurl's headline promise — "paste any
// curl command from the docs and it just works."
//
// Both the whitebox parser test (package gocurl) and the blackbox execution test
// (package tests) load this one manifest, so adding a documented command is a
// data-only append to compat.json — no new Go test function is required.
package corpus

import (
	_ "embed"
	"encoding/json"
)

//go:embed compat.json
var compatJSON []byte

// Expected is the parse result a Command must produce. Empty fields are not
// asserted, except Method which defaults to GET.
type Expected struct {
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	Headers     map[string]string   `json:"headers,omitempty"`
	Body        string              `json:"body,omitempty"`
	Query       map[string][]string `json:"query,omitempty"`
	BasicUser   string              `json:"basic_user,omitempty"`
	BasicPass   string              `json:"basic_pass,omitempty"`
	BearerToken string              `json:"bearer_token,omitempty"`
}

// Case is one corpus entry: a verbatim curl command and its expected parse.
type Case struct {
	Name    string   `json:"name"`
	Vendor  string   `json:"vendor"`
	Source  string   `json:"source"`
	Command string   `json:"command"`
	Want    Expected `json:"want"`
}

// Load returns every corpus case. A malformed manifest is a build/test error
// (panic), never a runtime path.
func Load() []Case {
	var cases []Case
	if err := json.Unmarshal(compatJSON, &cases); err != nil {
		panic("corpus: invalid compat.json: " + err.Error())
	}
	return cases
}
