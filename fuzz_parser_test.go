package gocurl

import (
	"errors"
	"testing"

	"github.com/maniartech/gocurl/internal/corpus"
	"github.com/maniartech/gocurl/tokenizer"
)

// FuzzParseCommand asserts the parse pipeline (preprocess -> tokenize -> convert)
// never panics for ANY input. Invariant: no panic; an error return is acceptable.
//
// Two measures keep the fuzz path safe and reproducible:
//   - It stubs convert.go's readFile seam, so the parse path is entirely
//     filesystem-free — no convert-time flag (-d @file, --data-urlencode
//     name@file, -b cookiefile) can be coaxed into reading an arbitrary or endless
//     host file (which would otherwise allocate unboundedly or hang, evading the
//     panic-only recover below).
//   - It deliberately SKIPS env-expansion. expandEnvInTokens reads the live process
//     environment, which would make inputs host-dependent (a committed crasher
//     might not reproduce elsewhere) and could inject a real host path. Variable
//     expansion is covered separately by deterministic unit tests.
//
// Run: go test -run=^$ -fuzz=FuzzParseCommand -fuzztime=30s .
func FuzzParseCommand(f *testing.F) {
	orig := readFile
	readFile = func(string) ([]byte, error) { return nil, errors.New("fuzz: filesystem disabled") }
	f.Cleanup(func() { readFile = orig })

	for _, c := range corpus.Load() {
		f.Add(c.Command)
	}
	for _, s := range []string{
		"curl 'unterminated quote",
		"curl -H 'X: ${UNCLOSED' https://x",
		"curl ${} $ $$ ${VAR https://x",
		"curl -X",
		"curl -H : https://x",
		"curl --max-time notanumber https://x",
		"curl -d a -d b -d c https://x",
		"curl -d @payload.json https://x",           // @file data (read stubbed)
		"curl -b cookies.txt https://x",             // cookie FILE (no '@'; read stubbed)
		"curl --data-urlencode name@file https://x", // urlencode @file (read stubbed)
		"",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, cmd string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic on %q: %v", cmd, r)
			}
		}()

		processed := preprocessMultilineCommand(cmd)
		tok := tokenizer.NewTokenizer()
		if tok.Tokenize(processed) != nil {
			return
		}
		// No env expansion (deterministic); file reads are stubbed (filesystem-free).
		_, _ = convertTokensToRequestOptions(tok.GetTokens())
	})
}
