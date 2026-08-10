package ante

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

func TestIsStructsMessage(t *testing.T) {
	require.True(t, IsStructsMessage("/structs.structs.MsgFleetMove"))
	require.True(t, IsStructsMessage("/structs.structs.MsgUpdateParams"))
	require.False(t, IsStructsMessage("/cosmos.bank.v1beta1.MsgSend"))
	require.False(t, IsStructsMessage(""))
	require.False(t, IsStructsMessage("/structs.structs.Msg"))
}

func TestIsFreeTransaction(t *testing.T) {
	// Use real proto message types since sdk.MsgTypeURL uses proto reflection
	t.Run("single structs msg", func(t *testing.T) {
		msgs := []sdk.Msg{&types.MsgFleetMove{}}
		require.True(t, IsFreeTransaction(msgs))
	})
	t.Run("multiple structs msgs", func(t *testing.T) {
		msgs := []sdk.Msg{&types.MsgFleetMove{}, &types.MsgStructAttack{}}
		require.True(t, IsFreeTransaction(msgs))
	})
	t.Run("MsgUpdateParams excluded", func(t *testing.T) {
		msgs := []sdk.Msg{&types.MsgUpdateParams{}}
		require.False(t, IsFreeTransaction(msgs))
	})
	t.Run("empty", func(t *testing.T) {
		require.False(t, IsFreeTransaction([]sdk.Msg{}))
	})
}

func TestKnownMessagesCompleteness(t *testing.T) {
	// Every message in PermissionMap must be in KnownStructsMessages
	for typeURL := range PermissionMap {
		require.True(t, KnownStructsMessages[typeURL], "PermissionMap entry %s not in KnownStructsMessages", typeURL)
	}

	// Every message in DynamicPermissionMessages must be in KnownStructsMessages
	for typeURL := range DynamicPermissionMessages {
		require.True(t, KnownStructsMessages[typeURL], "DynamicPermissionMessages entry %s not in KnownStructsMessages", typeURL)
	}

	// Every message in ChargeMessages must be in KnownStructsMessages
	for typeURL := range ChargeMessages {
		require.True(t, KnownStructsMessages[typeURL], "ChargeMessages entry %s not in KnownStructsMessages", typeURL)
	}

	// Every message in ProofMessages must be in KnownStructsMessages
	for typeURL := range ProofMessages {
		require.True(t, KnownStructsMessages[typeURL], "ProofMessages entry %s not in KnownStructsMessages", typeURL)
	}

	// Every message in SignatureMessages must be in KnownStructsMessages
	for typeURL := range SignatureMessages {
		require.True(t, KnownStructsMessages[typeURL], "SignatureMessages entry %s not in KnownStructsMessages", typeURL)
	}

	// Every message in ThrottleKeyExtractors must be in KnownStructsMessages
	for typeURL := range ThrottleKeyExtractors {
		require.True(t, KnownStructsMessages[typeURL], "ThrottleKeyExtractors entry %s not in KnownStructsMessages", typeURL)
	}
}

// TestEveryRegisteredMsgIsKnown is the guard against the failure mode where a
// new Msg is added to the proto and codec but nobody updates the ante maps: the
// message then registers fine, routes fine, and is rejected at the ante with
// "unknown structs message type" the first time anyone submits it. The maps can
// only be trusted if they are checked against the module's real message set
// rather than against each other.
func TestEveryRegisteredMsgIsKnown(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)

	registered := registry.ListImplementations(sdk.MsgInterfaceProtoName)
	require.NotEmpty(t, registered, "no structs messages registered; test would be vacuous")

	for _, typeURL := range registered {
		if !IsStructsMessage(typeURL) {
			continue
		}
		// MsgUpdateParams is a gov-gated paid tx, so it never reaches the
		// StructsDecorator (which only runs for free Structs txs).
		if typeURL == "/structs.structs.MsgUpdateParams" {
			continue
		}
		require.True(t, KnownStructsMessages[typeURL],
			"registered message %s is missing from KnownStructsMessages (update app/ante/maps.go)", typeURL)
	}
}

// TestEveryMessageHasCreatorAccessor guards the other half of the ante wiring
// that TestEveryRegisteredMsgIsKnown does not cover. A message declaring
// goproto_getters = false has no GetCreator(), so StructsDecorator can only
// find its signer through CreatorExtractors. Miss that entry and the message
// is rejected with "has no creator accessor" the first time anyone submits it,
// even though it is registered, routed, and present in every other ante map.
//
// The extractor is exercised rather than merely looked up: one that asserts to
// the wrong concrete type returns "" and fails as "type assertion failed",
// which points at the message instead of at the copy-pasted map entry.
func TestEveryMessageHasCreatorAccessor(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)

	registered := registry.ListImplementations(sdk.MsgInterfaceProtoName)
	require.NotEmpty(t, registered, "no structs messages registered; test would be vacuous")

	const sentinel = "structs1testcreatoraddress"

	for _, typeURL := range registered {
		if !IsStructsMessage(typeURL) || typeURL == MsgUpdateParamsTypeURL {
			continue
		}

		resolved, err := registry.Resolve(typeURL)
		require.NoError(t, err, "could not resolve registered message %s", typeURL)

		if _, ok := resolved.(creatorGetter); ok {
			continue
		}

		extractor, hasExtractor := CreatorExtractors[typeURL]
		require.True(t, hasExtractor,
			"%s has no GetCreator() and no CreatorExtractors entry, so the ante handler cannot find its signer (update app/ante/maps.go)", typeURL)

		creatorField := reflect.ValueOf(resolved).Elem().FieldByName("Creator")
		require.True(t, creatorField.IsValid() && creatorField.Kind() == reflect.String,
			"%s is in CreatorExtractors but has no string Creator field", typeURL)
		creatorField.SetString(sentinel)

		msg, ok := resolved.(sdk.Msg)
		require.True(t, ok, "%s does not implement sdk.Msg", typeURL)
		require.Equal(t, sentinel, extractor(msg),
			"CreatorExtractors entry for %s does not return the message's Creator (wrong concrete type in the assertion?)", typeURL)
	}
}

func TestEveryKnownMessageHasPermissionOrDynamic(t *testing.T) {
	for typeURL := range KnownStructsMessages {
		_, hasPerm := PermissionMap[typeURL]
		_, hasDyn := DynamicPermissionMessages[typeURL]
		require.True(t, hasPerm || hasDyn,
			"message %s is in KnownStructsMessages but has no entry in PermissionMap or DynamicPermissionMessages", typeURL)
	}
}

func TestProofMessagesHaveCorrectPermissions(t *testing.T) {
	expected := map[string]types.Permission{
		"/structs.structs.MsgStructBuildComplete":      types.PermHashBuild,
		"/structs.structs.MsgStructOreMinerComplete":   types.PermHashMine,
		"/structs.structs.MsgStructOreRefineryComplete": types.PermHashRefine,
		"/structs.structs.MsgPlanetRaidComplete":       types.PermHashRaid,
	}
	for typeURL, expectedPerm := range expected {
		actualPerm, ok := PermissionMap[typeURL]
		require.True(t, ok, "proof message %s missing from PermissionMap", typeURL)
		require.Equal(t, expectedPerm, actualPerm, "wrong permission for %s", typeURL)
	}
}

func TestPlayerUpdatePrimaryAddressRequiresPermAll(t *testing.T) {
	actualPerm, ok := PermissionMap["/structs.structs.MsgPlayerUpdatePrimaryAddress"]
	require.True(t, ok, "MsgPlayerUpdatePrimaryAddress missing from PermissionMap")
	require.Equal(t, types.PermAll, actualPerm,
		"primary address swap grants PermAll and must require it from the caller")
}

// AgreementOpen debits the player's primary address for the collateral. The access
// policy is dynamic and stays in the handler, but the spend bit applies under every
// policy, so it has to be enforced here too. A dynamic entry would make the
// decorator skip PermissionMap entirely and leave the ante with no check at all.
func TestAgreementOpenRequiresTokenTransfer(t *testing.T) {
	actualPerm, ok := PermissionMap["/structs.structs.MsgAgreementOpen"]
	require.True(t, ok, "MsgAgreementOpen missing from PermissionMap")
	require.Equal(t, types.PermTokenTransfer, actualPerm,
		"opening an agreement spends from the primary address and must require the token transfer bit")

	require.False(t, DynamicPermissionMessages["/structs.structs.MsgAgreementOpen"],
		"a dynamic entry would bypass the PermissionMap check above")
}

func TestAgreementDurationIncreaseRequiresTokenTransfer(t *testing.T) {
	actualPerm, ok := PermissionMap["/structs.structs.MsgAgreementDurationIncrease"]
	require.True(t, ok, "MsgAgreementDurationIncrease missing from PermissionMap")
	require.NotZero(t, actualPerm&types.PermTokenTransfer,
		"extending a duration buys blocks out of the primary address, so update rights alone are not enough")
	require.NotZero(t, actualPerm&types.PermUpdate,
		"the spend bit is additional to update rights on the agreement, not a replacement")
}

// TestArch_PrimaryAddressDebitsRequireTokenBit is the standing guard for the whole
// family of gaps the open-market bypass belonged to. A handler that debits the
// player's primary address is spending the player's money on the signer's
// authority, and the only control on that is a token bit on the signing key. An
// access or update permission is not a substitute: PermProviderOpen decides who
// may contract with a provider and PermUpdate who may modify an agreement, and
// neither says anything about whose coins may move.
//
// Rather than trusting review to notice the next one, this walks the handler
// sources and requires any that pairs GetPrimaryAddress with SendCoins to demand
// one of the asset bits in PermissionMap.
func TestArch_PrimaryAddressDebitsRequireTokenBit(t *testing.T) {
	// Handlers where the primary address is the destination rather than the
	// source. Sweeping another address of the same player into the primary is a
	// credit, so no spend authorization applies.
	creditsPrimary := map[string]string{
		"MsgAddressRegister": "sweeps the newly registered address into the primary",
		"MsgAddressRevoke":   "sweeps the revoked address into the primary",
	}

	// Bank calls that move coins out of an account. SendCoinsFromModuleToAccount
	// is the credit direction and is deliberately absent. Any other SendCoins*
	// method appearing in a handler fails the coverage check below rather than
	// quietly escaping this guard.
	debitCalls := map[string]bool{
		"SendCoins":                    true,
		"SendCoinsFromAccountToModule": true,
	}
	creditCalls := map[string]bool{
		"SendCoinsFromModuleToAccount": true,
	}
	seenSendCalls := map[string]bool{}

	keeperDir := filepath.Join("..", "..", "x", "structs", "keeper")
	entries, err := os.ReadDir(keeperDir)
	require.NoError(t, err)

	fset := token.NewFileSet()
	var failures []string
	checked := 0

	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "msg_server_") || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(keeperDir, name), nil, 0)
		require.NoError(t, err)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil {
				continue
			}

			msgName := handlerMessageType(fn)
			if msgName == "" {
				continue
			}

			var readsPrimary, debits bool
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				called := sel.Sel.Name
				if called == "GetPrimaryAddress" {
					readsPrimary = true
				}
				if strings.HasPrefix(called, "SendCoins") {
					seenSendCalls[called] = true
				}
				if debitCalls[called] {
					debits = true
				}
				return true
			})

			if !readsPrimary || !debits {
				continue
			}
			if _, exempt := creditsPrimary[msgName]; exempt {
				continue
			}

			checked++
			typeURL := "/structs.structs." + msgName

			if DynamicPermissionMessages[typeURL] {
				failures = append(failures, name+": "+msgName+
					" debits the primary address but is in DynamicPermissionMessages, which makes"+
					" StructsDecorator skip PermissionMap entirely")
				continue
			}

			perm, found := PermissionMap[typeURL]
			if !found {
				failures = append(failures, name+": "+msgName+
					" debits the primary address but has no PermissionMap entry")
				continue
			}
			if perm&types.PermAssetsAll == 0 {
				failures = append(failures, name+": "+msgName+
					" debits the primary address but requires no asset bit (has "+
					strconv.FormatUint(uint64(perm), 10)+")")
			}
		}
	}

	// A new way to move coins out of an account must be classified, or a handler
	// could debit the primary through it and never reach the check above.
	for called := range seenSendCalls {
		require.True(t, debitCalls[called] || creditCalls[called],
			"handlers use bankKeeper.%s, which this guard does not classify as a debit or a credit", called)
	}

	// If the scan silently stops matching, the guard is worthless but still green.
	require.GreaterOrEqual(t, checked, 4,
		"expected to find the known primary-address debits; the source scan is probably broken")

	require.Empty(t, failures,
		"handlers spending the player's primary address must require a token bit:\n  - %s",
		strings.Join(failures, "\n  - "))
}

// handlerMessageType returns the Msg type a msgServer method handles, taken from
// its *types.MsgX parameter, or "" if this is not a message handler.
func handlerMessageType(fn *ast.FuncDecl) string {
	if fn.Type.Params == nil {
		return ""
	}
	for _, param := range fn.Type.Params.List {
		star, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		sel, ok := star.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "types" || !strings.HasPrefix(sel.Sel.Name, "Msg") {
			continue
		}
		return sel.Sel.Name
	}
	return ""
}
