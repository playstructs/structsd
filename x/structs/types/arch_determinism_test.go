package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// determinismMarker is the escape hatch, mirroring the // SKIP_RATIONALE:
// convention that app/ante/arch_test.go uses for CheckTx short-circuits.
const determinismMarker = "DETERMINISM_OK:"

// toolchainUnicodeCalls are the calls whose answers come from the Unicode
// tables of whichever toolchain compiled the binary, keyed by the expression
// text the scan matches.
//
// unicode.Is is deliberately absent: unicode.Is(pinnedL, r) is a binary search
// over data we control and is exactly what the fix uses. What makes a call
// dangerous is naming one of the package's own tables, which is why the
// unicode.<Table> identifiers are listed separately below.
var toolchainUnicodeCalls = map[string]string{
	"unicode.IsLetter":   "classifies against the toolchain's letter table",
	"unicode.IsDigit":    "classifies against the toolchain's digit table",
	"unicode.IsNumber":   "classifies against the toolchain's number table",
	"unicode.IsSpace":    "classifies against the toolchain's White_Space table",
	"unicode.IsMark":     "classifies against the toolchain's mark tables",
	"unicode.IsPunct":    "classifies against the toolchain's punctuation table",
	"unicode.IsSymbol":   "classifies against the toolchain's symbol table",
	"unicode.IsControl":  "classifies against the toolchain's control table",
	"unicode.IsPrint":    "classifies against the toolchain's printable tables",
	"unicode.IsGraphic":  "classifies against the toolchain's graphic tables",
	"unicode.IsUpper":    "classifies against the toolchain's case tables",
	"unicode.IsLower":    "classifies against the toolchain's case tables",
	"unicode.IsTitle":    "classifies against the toolchain's case tables",
	"unicode.ToLower":    "maps case through the toolchain's case tables",
	"unicode.ToUpper":    "maps case through the toolchain's case tables",
	"unicode.ToTitle":    "maps case through the toolchain's case tables",
	"unicode.To":         "maps case through the toolchain's case tables",
	"unicode.SimpleFold": "folds through the toolchain's case orbit",

	"strings.ToLower":   "maps case through the toolchain's case tables",
	"strings.ToUpper":   "maps case through the toolchain's case tables",
	"strings.ToTitle":   "maps case through the toolchain's case tables",
	"strings.Title":     "maps case through the toolchain's case tables",
	"strings.EqualFold": "compares through the toolchain's case folding",
	"strings.TrimSpace": "trims against the toolchain's White_Space table",
	"strings.Fields":    "splits on the toolchain's White_Space table",

	"bytes.ToLower":   "maps case through the toolchain's case tables",
	"bytes.ToUpper":   "maps case through the toolchain's case tables",
	"bytes.EqualFold": "compares through the toolchain's case folding",
	"bytes.TrimSpace": "trims against the toolchain's White_Space table",
	"bytes.Fields":    "splits on the toolchain's White_Space table",
}

// toolchainUnicodeTables are the standard library's own range tables. Passing
// one to unicode.Is, or to anything else, reads the compiling toolchain.
var toolchainUnicodeTables = map[string]bool{
	"L": true, "Lu": true, "Ll": true, "Lt": true, "Lm": true, "Lo": true,
	"M": true, "Mn": true, "Mc": true, "Me": true,
	"N": true, "Nd": true, "Nl": true, "No": true,
	"P": true, "S": true, "Z": true, "Zs": true, "Zl": true, "Zp": true,
	"C": true, "Cc": true, "Cf": true, "Co": true, "Cs": true,
	"White_Space": true, "Letter": true, "Digit": true, "Space": true,
	"Categories": true, "Scripts": true, "Properties": true, "CaseRanges": true,
}

// regexpUnicodeClasses are the regexp constructs Go resolves against the
// compiling toolchain rather than against the pattern text.
var regexpUnicodeClasses = []string{
	`\p{`, `\P{`, `[[:alpha:]]`, `[[:upper:]]`, `[[:lower:]]`,
	`[[:alnum:]]`, `[[:space:]]`, `[[:punct:]]`, `[[:word:]]`, `(?i)`,
}

// TestArch_NoToolchainUnicodeInConsensusPaths keeps the determinism fix from
// eroding.
//
// Go resolves \p{L} in a regexp, and unicode.Is against unicode.L, using the
// Unicode tables of the toolchain that compiled the binary. Those tables grow
// with Go releases and nothing in this repository pins a toolchain hard enough
// to stop two validators from disagreeing. A code point one toolchain calls a
// letter and another calls unassigned decides a name validation differently on
// different nodes, and since only the accepting node writes the name, the two
// commit different state. The same applies to case folding, which is worse
// because NormalizeName's output is the guild name index key rather than just a
// comparison value.
//
// So consensus code classifies against the checked-in tables in
// unicode_tables.go instead. This test is what makes that a property of the
// package rather than a fact about the day it was written: a new
// strings.ToLower in a handler is a latent chain split, and it should fail here
// rather than in production.
//
// Two ways out for a legitimate use: an adjacent // DETERMINISM_OK: <reason>
// comment, or an entry in allowedFiles for a file that is wholly outside the
// state machine.
func TestArch_NoToolchainUnicodeInConsensusPaths(t *testing.T) {
	// Files exempt from the check, each with the reason it cannot affect
	// consensus. Keep this short: an entry here is a file where a Unicode table
	// difference is asserted not to reach state.
	allowedFiles := map[string]string{
		// The generated tables and their generator are the pinned source of
		// truth; they necessarily name the standard library's tables in order to
		// copy them.
		"x/structs/types/unicode_tables.go":           "the generated pinned tables themselves",
		"x/structs/types/internal/maketables/main.go": "the generator, which reads the toolchain by design",
		// unicode_pinned.go is the pinned implementation. It mirrors
		// unicode.IsSpace and unicode.ToLower against our own tables, so it
		// mentions their names in prose and uses unicode.Is with pinned tables.
		"x/structs/types/unicode_pinned.go": "the pinned implementation, which reads only our tables",
		// autocli and the CLI build help text and parse operator input. Neither
		// reaches the state machine.
		"x/structs/module/autocli.go": "CLI wiring, never executed during block processing",
	}

	root := repoRoot(t)

	// Directories that make up the deterministic state machine.
	scanDirs := []string{
		filepath.Join(root, "x", "structs"),
		filepath.Join(root, "app"),
	}

	fset := token.NewFileSet()
	var failures []string
	filesScanned := 0

	for _, dir := range scanDirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			rel, relErr := filepath.Rel(root, path)
			require.NoError(t, relErr)
			rel = filepath.ToSlash(rel)

			if _, exempt := allowedFiles[rel]; exempt {
				return nil
			}

			src, readErr := os.ReadFile(path)
			require.NoError(t, readErr)

			file, parseErr := parser.ParseFile(fset, path, src, parser.ParseComments)
			require.NoError(t, parseErr)
			filesScanned++

			lines := strings.Split(string(src), "\n")

			// annotated reports whether a DETERMINISM_OK comment sits on the
			// offending line or the two lines above it.
			annotated := func(line int) bool {
				for i := line - 3; i < line && i < len(lines); i++ {
					if i >= 0 && strings.Contains(lines[i], determinismMarker) {
						return true
					}
				}
				return false
			}

			report := func(pos token.Position, what, why string) {
				if annotated(pos.Line) {
					return
				}
				failures = append(failures, rel+":"+itoa(pos.Line)+": "+what+" "+why)
			}

			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				qualified := pkg.Name + "." + sel.Sel.Name

				if why, bad := toolchainUnicodeCalls[qualified]; bad {
					report(fset.Position(sel.Pos()), qualified, why)
					return true
				}
				if pkg.Name == "unicode" && toolchainUnicodeTables[sel.Sel.Name] {
					report(fset.Position(sel.Pos()), qualified,
						"is one of the toolchain's own range tables; use the pinned table instead")
				}
				return true
			})

			// Regexp Unicode classes live in string literals, so they need a
			// separate pass over the literal values rather than the call graph.
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				for _, class := range regexpUnicodeClasses {
					if strings.Contains(lit.Value, class) {
						report(fset.Position(lit.Pos()), lit.Value,
							"contains "+class+", which regexp resolves against the toolchain's Unicode tables")
						break
					}
				}
				return true
			})

			return nil
		})
		require.NoError(t, err)
	}

	// A scan that silently stops finding files would pass forever. The repo has
	// hundreds of non-test sources under these two trees.
	require.Greater(t, filesScanned, 100,
		"only scanned %d files; the walk is probably broken", filesScanned)

	require.Empty(t, failures,
		"consensus code must classify against the pinned Unicode tables in unicode_tables.go, "+
			"not the compiling toolchain's. Either use the pinned helpers in unicode_pinned.go, "+
			"or add an adjacent `// %s <reason>` comment if the value provably cannot reach state:\n  - %s",
		determinismMarker, strings.Join(failures, "\n  - "))
}

// repoRoot walks up from the test's working directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "walked to the filesystem root without finding go.mod")
		dir = parent
	}
}

// itoa avoids pulling strconv in for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
