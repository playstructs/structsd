package keeper_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A new infusion mutation must keep base power and the owner portfolio together.
// Behavioral tests check the arithmetic; this guard prevents new callers from
// bypassing that boundary or publishing derived rows before cache commit.
func TestArch_DiversityMutationBoundary(t *testing.T) {
	allowed := map[string]map[string]bool{
		"ApplyOwnerInfusionCapacityDelta": {"InfusionCache.applyPlayerCapacityDelta": true},
		"ChangeReactorDiversityPosition": {
			"CurrentContext.ApplyOwnerInfusionCapacityDelta": true,
			"Keeper.RebuildReactorDiversity":                 true, // explicit full reconstruction
		},
		"setReactorDiversityPosition": {"ReactorDiversityPositionCache.Commit": true},
		"setOwnerReactorDiversity":    {"OwnerReactorDiversityCache.Commit": true},
	}
	seen := make(map[string]map[string]bool)
	files, err := os.ReadDir(".")
	require.NoError(t, err)
	fset := token.NewFileSet()
	for _, entry := range files {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			receiver := ""
			if fn.Recv != nil {
				expr := fn.Recv.List[0].Type
				if ptr, ok := expr.(*ast.StarExpr); ok {
					expr = ptr.X
				}
				if id, ok := expr.(*ast.Ident); ok {
					receiver = id.Name
				}
			}
			caller := receiver + "." + fn.Name.Name
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				method := selector.Sel.Name
				if callers, restricted := allowed[method]; restricted {
					require.True(t, callers[caller], "%s: %s bypasses the diversity mutation boundary through %s", fset.Position(call.Pos()), caller, method)
					if seen[method] == nil {
						seen[method] = make(map[string]bool)
					}
					seen[method][caller] = true
				}
				if receiver == "InfusionCache" && (method == "SetGridAttribute" || method == "SetGridAttributeDelta") {
					if fn.Name.Name == "applyPlayerCapacityDelta" {
						return true // base-only initialization before diversity reconstruction
					}
					require.Equal(t, "applyGridDeltas", fn.Name.Name, "%s: infusion grid writes belong to the contribution helpers", fset.Position(call.Pos()))
					require.NotEmpty(t, call.Args)
					attribute, ok := call.Args[0].(*ast.SelectorExpr)
					require.True(t, ok, "%s: direct infusion writes may target only destination attributes", fset.Position(call.Pos()))
					require.Contains(t, []string{"DestinationFuelAttributeId", "DestinationCapacityAttributeId"}, attribute.Sel.Name)
				}
				return true
			})
		}
	}
	for method, callers := range allowed {
		require.Equal(t, callers, seen[method], "remove stale allowances or restore the missing %s integration", method)
	}
}

func TestArch_DiversityRefreshPrecedesCacheCommits(t *testing.T) {
	_, file := parseCurrentContext(t)
	fn := findCommitAll(t, file)
	refresh := token.NoPos
	var commits []token.Pos
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "refreshDirtyReactorDiversity" {
			refresh = call.Pos()
		}
		if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "commitCaches" {
			commits = append(commits, call.Pos())
		}
		return true
	})
	require.NotEqual(t, token.NoPos, refresh, "dirty portfolios must be refreshed before publishing dependent capacity")
	require.NotEmpty(t, commits)
	for _, position := range commits {
		require.Less(t, refresh, position)
	}
}
