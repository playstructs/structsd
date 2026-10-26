package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The exception for GO-2026-5970 depends on not introducing norm.Iter into our
// input paths. Resolve import aliases so renaming the import cannot bypass it.
func TestArch_NoPinnedNormalizationIterator(t *testing.T) {
	root := repoRoot(t)
	for _, dir := range []string{"app", "cmd", "x"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			for _, imp := range f.Imports {
				imported, err := strconv.Unquote(imp.Path.Value)
				require.NoError(t, err)
				if imported != "golang.org/x/text/unicode/norm" {
					continue
				}
				name := "norm"
				if imp.Name != nil {
					name = imp.Name.Name
				}
				require.NotEqual(t, ".", name, "%s: normalization imports must be auditable", path)
				ast.Inspect(f, func(node ast.Node) bool {
					sel, ok := node.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					pkg, ok := sel.X.(*ast.Ident)
					if ok && pkg.Name == name && sel.Sel.Name == "Iter" {
						t.Errorf("%s: norm.Iter requires a new GO-2026-5970 input-safety review while x/text is pinned", path)
					}
					return true
				})
			}
			return nil
		})
		require.NoError(t, err)
	}
}
