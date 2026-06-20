package gocurl_test

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestAPISurface is the v1 public-surface guard. It enumerates every exported
// declaration of the root gocurl package and compares it to the committed golden
// api.txt. Any addition, removal, or signature change to the exported surface
// fails this test until api.txt (and the CHANGELOG) are updated deliberately.
//
// Regenerate the golden after an intentional surface change:
//
//	GOCURL_UPDATE_API=1 go test -run TestAPISurface .
//
// The guard is dependency-free (stdlib go/ast only) so it runs in the normal
// `go test` job; it does not freeze the surface for v1 yet (gocurl is pre-tag),
// it just makes every change to it explicit and reviewable.
func TestAPISurface(t *testing.T) {
	got := collectAPISurface(t)

	const golden = "api.txt"
	if os.Getenv("GOCURL_UPDATE_API") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d lines)", golden, strings.Count(got, "\n"))
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read %s: %v (run `GOCURL_UPDATE_API=1 go test -run TestAPISurface .` to create it)", golden, err)
	}
	if string(want) != got {
		t.Errorf("exported API surface changed.\nIf intentional, update the golden + CHANGELOG:\n"+
			"  GOCURL_UPDATE_API=1 go test -run TestAPISurface .\n\n%s", diff(string(want), got))
	}
}

// collectAPISurface parses the root package's non-test sources and returns a
// sorted, newline-joined list of its exported declarations (funcs with
// signatures, exported-receiver methods, and types/vars/consts by name).
func collectAPISurface(t *testing.T) string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs["gocurl"]
	if pkg == nil {
		t.Fatal("package gocurl not found in .")
	}

	render := func(n ast.Node) string {
		var sb strings.Builder
		_ = printer.Fprint(&sb, fset, n)
		return sb.String()
	}

	var lines []string
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				sig := strings.TrimPrefix(render(d.Type), "func") // "(params) results"
				if d.Recv != nil && len(d.Recv.List) == 1 {
					recv := render(d.Recv.List[0].Type)
					if !ast.IsExported(strings.TrimPrefix(recv, "*")) {
						continue
					}
					lines = append(lines, "func ("+recv+") "+d.Name.Name+sig)
				} else if d.Recv == nil {
					lines = append(lines, "func "+d.Name.Name+sig)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							lines = append(lines, "type "+s.Name.Name)
						}
					case *ast.ValueSpec:
						kind := "var"
						if d.Tok == token.CONST {
							kind = "const"
						}
						for _, name := range s.Names {
							if name.IsExported() {
								lines = append(lines, kind+" "+name.Name)
							}
						}
					}
				}
			}
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

// diff returns a minimal line-level added/removed summary.
func diff(want, got string) string {
	w := map[string]bool{}
	for _, l := range strings.Split(want, "\n") {
		w[l] = true
	}
	g := map[string]bool{}
	for _, l := range strings.Split(got, "\n") {
		g[l] = true
	}
	var b strings.Builder
	for _, l := range strings.Split(got, "\n") {
		if l != "" && !w[l] {
			b.WriteString("  + " + l + "\n")
		}
	}
	for _, l := range strings.Split(want, "\n") {
		if l != "" && !g[l] {
			b.WriteString("  - " + l + "\n")
		}
	}
	return b.String()
}
