package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

/* Dependency floors that are consensus properties rather than preferences.
 *
 * Read from go.mod rather than the binary: test binaries carry no dependency
 * list in their build info, and in a tidy module under graph pruning (go 1.17+)
 * the require line for every module in the build list is the version MVS
 * selected, so go.mod is the linked version as long as `go mod tidy` has run.
 *
 * A floor goes here when dropping below it would reintroduce a halt, a fork or
 * a node-local divergence - not for features. Raise the require in go.mod
 * rather than lowering a floor. A floor whose fix changes which transactions
 * or packets the chain accepts is consensus-affecting and has to ship in a
 * coordinated upgrade, never as a rolling binary swap.
 */

type dependencyFloor struct {
	path   string
	floor  string
	reason string
}

var dependencyFloors = []dependencyFloor{
	{
		// Below v1.2.7, nodeDB.DeleteFastNode evicted a key from the in-memory
		// fast-node cache before the batch that deleted it from disk was
		// written, and released the mutex in between. A concurrent Get on that
		// key - gRPC and LCD queries run against the query router without the
		// ABCI mutex, so they overlap Commit - missed the cache, read the
		// not-yet-deleted node from disk and put it back. After Commit the disk
		// was right and the cache held a deleted key for good.
		//
		// The fast-node index is not hashed, so AppHash stayed in agreement
		// while a handler that read the key took a branch nobody else took:
		// extra events and extra gas, a LastResultsHash the network did not
		// have, and CONSENSUS FAILURE on the next block (incident 2026-09,
		// structs-public-rpc, height 2586840). Only nodes serving queries at
		// volume are exposed, which is why it presented as a recurring "public
		// node stall" and why a rollback always cleared it: the stale value
		// lived in process memory.
		path:  "github.com/cosmos/iavl",
		floor: "v1.2.7",
		reason: "fast-node cache/commit race (cosmos/iavl#1142): a node serving gRPC/LCD queries can cache a " +
			"deleted key and read it back inside a transaction, diverging on LastResultsHash while AppHash still agrees",
	},
	{
		// v10.0.0 shipped four days before the v10 line picked up
		// ASA-2025-004 (GHSA-jg6f-48ff-5xrw): AcknowledgePacket accepted an
		// acknowledgement whose JSON did not round-trip through the canonical
		// encoder, and non-deterministic unmarshalling of it could halt the
		// chain. v10.1.0 rejects an acknowledgement whose re-encoding differs
		// from the bytes supplied. Channel handshakes are permissionless, so
		// the exposure does not depend on a channel existing today.
		path:  "github.com/cosmos/ibc-go/v10",
		floor: "v10.1.0",
		reason: "ASA-2025-004 acknowledgement round-trip check in AcknowledgePacket; below it a crafted IBC " +
			"acknowledgement can halt the chain",
	},
	{
		// v0.53.8 bounds the indices in multisig verification and tx signature
		// decoding so arbitrary transaction bytes return errors instead of
		// panicking, validates the SEC1 tag byte of a compressed secp256k1
		// pubkey, and stops x/distribution erroring inside Begin/EndBlock when
		// a withdraw address is blocked or a historical-rewards record is
		// absent. The pubkey check changes which transactions are valid.
		path:  "github.com/cosmos/cosmos-sdk",
		floor: "v0.53.8",
		reason: "tx-decode and multisig panics, secp256k1 SEC1 tag validation, x/distribution block-hook errors " +
			"(SDK v0.53.8)",
	},
	{
		// v0.38.25 (v0.38.24 was skipped) adds MsgBytesFilter to the mempool
		// reactor, closing a heap-amplification DoS from crafted gossip, and
		// fixes the setRecheckFull/setDone race that produced spurious
		// ErrRecheckFull - the shape of this repo's stuck-mempool incidents.
		// Engine-only, no app-state effect.
		path:  "github.com/cometbft/cometbft",
		floor: "v0.38.25",
		reason: "mempool heap-amplification filter and the ErrRecheckFull race (CometBFT v0.38.25)",
	},
}

func TestArch_DependencyFloors(t *testing.T) {
	root := moduleRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	require.NoError(t, err)

	mod, err := modfile.Parse("go.mod", data, nil)
	require.NoError(t, err)

	resolved := make(map[string]string)
	for _, req := range mod.Require {
		resolved[req.Mod.Path] = req.Mod.Version
	}
	for _, rep := range mod.Replace {
		resolved[rep.Old.Path] = rep.New.Version
	}

	for _, floor := range dependencyFloors {
		t.Run(floor.path, func(t *testing.T) {
			version, ok := resolved[floor.path]
			require.True(t, ok, "%s is not in go.mod; the layer this floor guards changed underneath the test", floor.path)
			require.True(t, semver.IsValid(version),
				"%s version %q is not semver; a pseudo-version or local replace needs an explicit review against %s",
				floor.path, version, floor.floor)

			require.GreaterOrEqual(t, semver.Compare(version, floor.floor), 0,
				"%s resolved to %s, below %s.\n"+
					"That version has: %s.\n"+
					"Raise the require in go.mod rather than lowering this floor; a store, SDK or IBC bump that pulls "+
					"the module back down reintroduces the incident.",
				floor.path, version, floor.floor, floor.reason)
		})
	}
}

// moduleRoot walks up from the test's working directory to the module root.
func moduleRoot(t *testing.T) string {
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
