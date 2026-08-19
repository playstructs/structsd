package keeper_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests guard the commit path against Go map iteration order.
//
// Ranging a cache map directly is only safe while every Commit writes to keys
// derived from its own map key. AddressCache.Commit does not: it allocates an
// auth account number from the account keeper's global sequence for any address
// without one. Committing two such addresses in map order gave two nodes
// different account numbers for the same pair, which is divergent auth state
// and a different app hash. It was reachable from a genesis AddressList and
// from one transaction carrying two AddressRegister messages.
//
// commitCaches sorts, which makes the order a property of the data rather than
// of the runtime. The point of these tests is that the property survives the
// next person to add a cache.

const commitAllFuncName = "CommitAll"

// parseCurrentContext returns the AST for the file that owns CommitAll.
func parseCurrentContext(t *testing.T) (*token.FileSet, *ast.File) {
	t.Helper()

	path := "current_context.go"
	source, err := os.ReadFile(path)
	require.NoError(t, err, "reading %s", path)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	require.NoError(t, err, "parsing %s", path)

	return fset, file
}

// findCommitAll returns the CommitAll function declaration.
func findCommitAll(t *testing.T, file *ast.File) *ast.FuncDecl {
	t.Helper()

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != commitAllFuncName || funcDecl.Recv == nil {
			continue
		}
		return funcDecl
	}

	t.Fatalf("could not find %s in current_context.go", commitAllFuncName)
	return nil
}

// cacheMapFields returns the names of every map-typed field on CurrentContext,
// which is the set of caches that exist.
func cacheMapFields(t *testing.T, file *ast.File) []string {
	t.Helper()

	var fields []string

	ast.Inspect(file, func(node ast.Node) bool {
		typeSpec, ok := node.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != "CurrentContext" {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}

		for _, field := range structType.Fields.List {
			if _, isMap := field.Type.(*ast.MapType); !isMap {
				continue
			}
			for _, name := range field.Names {
				fields = append(fields, name.Name)
			}
		}
		return false
	})

	require.NotEmpty(t, fields, "found no map fields on CurrentContext; the parser is looking in the wrong place")
	sort.Strings(fields)
	return fields
}

// TestArch_CommitAllDoesNotRangeCacheMaps fails on a bare range over a cache
// map inside CommitAll. That is the exact shape the bug had, and it is the
// shape someone adding a nineteenth cache would copy from its neighbours.
func TestArch_CommitAllDoesNotRangeCacheMaps(t *testing.T) {
	fset, file := parseCurrentContext(t)
	commitAll := findCommitAll(t, file)
	cacheFields := cacheMapFields(t, file)

	isCacheField := make(map[string]bool, len(cacheFields))
	for _, field := range cacheFields {
		isCacheField[field] = true
	}

	var offenders []string

	ast.Inspect(commitAll.Body, func(node ast.Node) bool {
		rangeStmt, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}

		selector, ok := rangeStmt.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !isCacheField[selector.Sel.Name] {
			return true
		}

		offenders = append(offenders, fset.Position(rangeStmt.Pos()).String()+": range over cc."+selector.Sel.Name)
		return true
	})

	require.Empty(t, offenders,
		"CommitAll must commit through commitCaches, which sorts. Go randomizes map iteration order, "+
			"and AddressCache.Commit allocates an auth account number from a global sequence, so a bare "+
			"range here gives two validators different auth state and a different app hash:\n%s",
		strings.Join(offenders, "\n"))
}

// TestArch_CommitAllCommitsEveryCacheMap catches the other half of the mistake:
// a cache map that CommitAll never commits at all. Nothing else in the suite
// notices that, and the symptom is silently discarded writes rather than a
// crash.
func TestArch_CommitAllCommitsEveryCacheMap(t *testing.T) {
	_, file := parseCurrentContext(t)
	commitAll := findCommitAll(t, file)
	cacheFields := cacheMapFields(t, file)

	committed := make(map[string]bool)

	ast.Inspect(commitAll.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "commitCaches" || len(call.Args) != 1 {
			return true
		}

		if selector, ok := call.Args[0].(*ast.SelectorExpr); ok {
			committed[selector.Sel.Name] = true
		}
		return true
	})

	// structTypes is read-only and deliberately never committed; its Commit is
	// an empty method. Everything else must be committed.
	exempt := map[string]string{
		"structTypes": "read-only cache, StructTypeCache.Commit is intentionally empty",
	}

	var missing []string
	for _, field := range cacheFields {
		if committed[field] || exempt[field] != "" {
			continue
		}
		missing = append(missing, "cc."+field)
	}

	require.Empty(t, missing,
		"every cache map on CurrentContext must be passed to commitCaches in CommitAll, or added to the "+
			"exempt list with a reason. These are never committed, so their writes are silently dropped:\n%s",
		strings.Join(missing, "\n"))
}

// TestArch_AccountCreationStaysOutOfCommitPaths keeps the account-number
// sequence where it can be reasoned about.
//
// NewAccountWithAddress is the only call in the module that consumes a global
// monotonic sequence, which makes it the only thing whose result depends on
// when it runs. The three allowlisted sites are safe for different reasons:
// address.go is reached from a commit and is the reason commitCaches sorts,
// while guild.go and provider_context.go are called directly from handlers, one
// per message. A fourth site would need the same analysis, so it has to be a
// deliberate edit here rather than something that slips in.
func TestArch_AccountCreationStaysOutOfCommitPaths(t *testing.T) {
	allowed := map[string]string{
		"address.go":          "SetPlayerIndexForAddress; reached from AddressCache.Commit, which is why commitCaches sorts",
		"guild.go":            "AppendGuild; called once per MsgGuildCreate",
		"provider_context.go": "NewProvider; called once per MsgProviderCreate",
	}

	var offenders []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(source), "NewAccountWithAddress") {
			return nil
		}

		if allowed[filepath.Base(path)] == "" {
			offenders = append(offenders, path)
		}
		return nil
	})
	require.NoError(t, err)

	require.Empty(t, offenders,
		"NewAccountWithAddress consumes the account keeper's global account-number sequence, so its result "+
			"depends on call order. Adding a call site means proving that site runs in a deterministic order. "+
			"Allowlist it in this test once you have:\n%s",
		strings.Join(offenders, "\n"))

	// A stale allowlist is as bad as a missing one: if a site moves away, the
	// entry should go too rather than quietly permitting a future file of the
	// same name.
	for name := range allowed {
		source, err := os.ReadFile(name)
		require.NoError(t, err, "allowlisted %s no longer exists; remove it from this test", name)
		require.Contains(t, string(source), "NewAccountWithAddress",
			"%s is allowlisted but no longer creates accounts; remove it from this test", name)
	}
}
