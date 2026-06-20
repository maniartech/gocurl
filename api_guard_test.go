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
	// Guard both packages that carry public surface a caller depends on: the root
	// engine AND the options package (the typed builder + RequestOptions). Covering
	// options is deliberate — when a removal orphaned the builder from its executor,
	// a root-only guard could not catch it; now a change to either fails here.
	checkSurface(t, ".", "gocurl", "api.txt")
	checkSurface(t, "options", "options", "api_options.txt")
}

// checkSurface compares one package's live exported surface to its golden file,
// regenerating the golden instead when GOCURL_UPDATE_API=1.
func checkSurface(t *testing.T, dir, pkgName, golden string) {
	t.Helper()
	got := collectAPISurface(t, dir, pkgName)

	if os.Getenv("GOCURL_UPDATE_API") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d lines)", golden, strings.Count(got, "\n"))
		return
	}

	wantBytes, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read %s: %v (run `GOCURL_UPDATE_API=1 go test -run TestAPISurface .` to create it)", golden, err)
	}
	// Normalize CRLF so a Windows checkout (core.autocrlf=true) doesn't flake the
	// byte-exact compare; .gitattributes also pins *.txt to LF.
	want := strings.ReplaceAll(string(wantBytes), "\r\n", "\n")
	if want != got {
		t.Errorf("exported API surface of package %q changed.\nIf intentional, update the golden + CHANGELOG:\n"+
			"  GOCURL_UPDATE_API=1 go test -run TestAPISurface .\n\n%s", pkgName, diff(want, got))
	}
}

// embeddedName returns the type name of an embedded struct field (T, *T, or
// pkg.T), or "" if it cannot be determined.
func embeddedName(t ast.Expr) string {
	switch e := t.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return embeddedName(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

// collectAPISurface parses pkgName's non-test sources under dir and returns a
// sorted, newline-joined list of its exported declarations (funcs with
// signatures, exported-receiver methods, and types/vars/consts by name).
func collectAPISurface(t *testing.T, dir, pkgName string) string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[pkgName]
	if pkg == nil {
		t.Fatalf("package %q not found in %s", pkgName, dir)
	}

	render := func(n ast.Node) string {
		var sb strings.Builder
		_ = printer.Fprint(&sb, fset, n)
		return sb.String()
	}

	// typeBody renders a type's definition so the guard catches member changes
	// (interface methods, func-type signatures, exported struct fields), not just
	// the name. For structs only the EXPORTED fields are kept — a change to an
	// unexported field is not a public-API change and would only add churn.
	typeBody := func(t ast.Expr) string {
		st, ok := t.(*ast.StructType)
		if !ok || st.Fields == nil {
			return render(t)
		}
		var kept []*ast.Field
		for _, f := range st.Fields.List {
			if len(f.Names) == 0 { // embedded field
				if n := embeddedName(f.Type); n != "" && ast.IsExported(n) {
					kept = append(kept, f)
				}
				continue
			}
			var names []*ast.Ident
			for _, n := range f.Names {
				if n.IsExported() {
					names = append(names, n)
				}
			}
			if len(names) > 0 {
				nf := *f
				nf.Names = names
				kept = append(kept, &nf)
			}
		}
		clone := *st
		clone.Fields = &ast.FieldList{List: kept}
		return render(&clone)
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
							lines = append(lines, "type "+s.Name.Name+" "+typeBody(s.Type))
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
