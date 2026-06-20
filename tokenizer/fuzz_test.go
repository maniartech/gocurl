package tokenizer

import "testing"

// FuzzTokenize asserts the tokenizer never panics, loops forever, or allocates
// unboundedly for ANY input. The tokenizer is pure (no I/O), so it is safe to
// fuzz the full surface. An error return is an acceptable outcome; a panic is not.
//
// Run: go test -run=^$ -fuzz=FuzzTokenize -fuzztime=30s ./tokenizer/
func FuzzTokenize(f *testing.F) {
	seeds := []string{
		`curl -X POST -d '{"k":"v"}' https://api.example.com`,
		`curl -H 'Authorization: Bearer $TOK' $URL/data`,
		`curl -u user:pass https://example.com`,
		`curl --data-binary @payload.json https://x`,
		`curl -H "X: y" -H "Z:`, // unterminated header value
		`curl 'unterminated quote`,
		`curl "a\"b" 'c'd' https://x`,         // mixed/escaped quotes
		"curl \\\n  -X PATCH \\\n  https://x", // line continuations
		"# comment only",
		"curl ${VAR} $ $$ ${ https://x", // odd variable syntax
		"",
		"curl",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, cmd string) {
		tok := NewTokenizer()
		_ = tok.Tokenize(cmd) // must never panic; an error is fine
		_ = tok.GetTokens()
	})
}
