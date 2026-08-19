package keeper_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArch_TeardownPathsCheckpointBeforeMutating enforces the ordering that every
// agreement settlement depends on.
//
// ProviderCache.Checkpoint bills the provider's *current* agreement load across
// the whole span since the last checkpoint. Once a teardown has decremented the
// load and removed the agreement, no later checkpoint can account for the span
// that agreement was live: the revenue it earned stays in the collateral pool,
// and neither Checkpoint nor WithdrawBalanceAndCommit can reach it because both
// are computed from load. Checkpointing after settlement therefore does not
// merely misorder two writes, it strands money permanently.
//
// The same ordering protects the other direction on the paths that re-base an
// agreement: change the capacity or the window before checkpointing and the new
// load gets billed across the old span.
//
// This was filed as a bug against PrematureCloseByAllocation, which was the one
// path that settled without checkpointing while the other three did it. That is
// the shape the rule has to prevent: not a wrong implementation, but one path
// out of several quietly disagreeing with its siblings. So rather than name the
// paths, this finds them by their own marker — beginTeardown, which claims the
// single settlement an agreement gets — and holds every one of them to the rule.
func TestArch_TeardownPathsCheckpointBeforeMutating(t *testing.T) {
	const (
		teardownMarker = "beginTeardown"
		checkpointCall = "checkpointProvider"
	)

	// Everything that must not happen before the provider has been checkpointed:
	// the payouts, the load change, the removal, and the window re-base.
	mutations := map[string]string{
		"PayoutProviderCancellationPenalty":                    "pays the consumer out of the collateral pool",
		"PayoutConsumerCancellationPenaltyAndReturnCollateral": "pays the consumer and sweeps the penalty as revenue",
		"PayoutVoidedProviderCancellationPenalty":              "sweeps the penalty to the provider",
		"ReturnRemainingCollateral":                            "pays the consumer out of the collateral pool",
		"AgreementLoadDecrease":                                "changes the load Checkpoint bills against",
		"removeAgreement":                                      "takes the agreement out of state",
		"SetStartBlock":                                        "re-bases the span Checkpoint bills across",
		"ResetStartBlock":                                      "re-bases the span Checkpoint bills across",
	}

	path := "agreement_cache.go"
	source, err := os.ReadFile(path)
	require.NoError(t, err, "reading %s", path)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	require.NoError(t, err, "parsing %s", path)

	// firstCall returns the source position of the earliest call to name inside
	// body, or -1. Positions are compared rather than traversal order so the
	// result is the order a reader sees.
	firstCall := func(body *ast.BlockStmt, name string) token.Pos {
		earliest := token.NoPos

		ast.Inspect(body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			var called string
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				called = fun.Name
			case *ast.SelectorExpr:
				called = fun.Sel.Name
			}

			if called != name {
				return true
			}
			if earliest == token.NoPos || call.Pos() < earliest {
				earliest = call.Pos()
			}
			return true
		})

		return earliest
	}

	var found []string

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || funcDecl.Body == nil {
			continue
		}

		// Only the settlement paths, identified by the claim they all make.
		if firstCall(funcDecl.Body, teardownMarker) == token.NoPos {
			continue
		}
		found = append(found, funcDecl.Name.Name)

		checkpointAt := firstCall(funcDecl.Body, checkpointCall)
		require.NotEqual(t, token.NoPos, checkpointAt,
			"%s claims a settlement with %s but never calls %s. Checkpoint bills the current load "+
				"across the span since the last checkpoint, so a path that settles without one leaves the "+
				"revenue earned over its own lifetime stranded in the collateral pool with no way to reclaim it.",
			funcDecl.Name.Name, teardownMarker, checkpointCall)

		for mutation, why := range mutations {
			mutationAt := firstCall(funcDecl.Body, mutation)
			if mutationAt == token.NoPos {
				continue
			}

			require.Less(t, int(checkpointAt), int(mutationAt),
				"%s calls %s at %s, before %s at %s. %s, so the checkpoint has to come first.",
				funcDecl.Name.Name, mutation, fset.Position(mutationAt), checkpointCall,
				fset.Position(checkpointAt), why)
		}
	}

	sort.Strings(found)
	require.Equal(t,
		[]string{"Expire", "PrematureCloseByAllocation", "PrematureCloseByConsumer", "PrematureCloseByProvider"},
		found,
		"the set of settlement paths changed. A new one is fine, but add it here deliberately: this test is "+
			"the only thing that stops it settling without a checkpoint, which is how the allocation-driven "+
			"path came to strand provider revenue.")
}
