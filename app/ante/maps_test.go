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

	/* Founding a guild is a race for one global puzzle, and the throttle only
	 * caps one attempt per signer per block. Free attempts would make flooding
	 * that race from a spread of throwaway signers cost nothing at all, so a
	 * losing proof has to cost gas.
	 */
	t.Run("MsgGuildCreate pays", func(t *testing.T) {
		msgs := []sdk.Msg{&types.MsgGuildCreate{}}
		require.False(t, IsFreeTransaction(msgs))
		require.False(t, IsAnyFreeTransaction(msgs))
	})

	// And it cannot be smuggled onto the free path by pairing it with one.
	t.Run("MsgGuildCreate poisons a free batch", func(t *testing.T) {
		msgs := []sdk.Msg{&types.MsgFleetMove{}, &types.MsgGuildCreate{}}
		require.False(t, IsFreeTransaction(msgs))
	})

	t.Run("MsgReactorRestart pays", func(t *testing.T) {
		msgs := []sdk.Msg{&types.MsgReactorRestart{}}
		require.False(t, IsFreeTransaction(msgs))
		require.False(t, IsAnyFreeTransaction(msgs))
	})

	t.Run("MsgReactorRestart poisons a free batch", func(t *testing.T) {
		msgs := []sdk.Msg{&types.MsgFleetMove{}, &types.MsgReactorRestart{}}
		require.False(t, IsFreeTransaction(msgs))
	})
}

// Every priced message still has to be a Structs message, or IsFreeTransaction
// would have refused it for the wrong reason and the entry would be decoration.
func TestPricedStructsMessagesAreKnown(t *testing.T) {
	for typeURL := range PricedStructsMessages {
		require.True(t, KnownStructsMessages[typeURL], "PricedStructsMessages entry %s not in KnownStructsMessages", typeURL)
	}
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
		require.NotEqual(t, hasPerm, hasDyn,
			"message %s must be in exactly one of PermissionMap or DynamicPermissionMessages", typeURL)
	}
}

func TestProofMessagesHaveCorrectPermissions(t *testing.T) {
	expected := map[string]types.Permission{
		"/structs.structs.MsgStructBuildComplete":       types.PermHashBuild,
		"/structs.structs.MsgStructOreMinerComplete":    types.PermHashMine,
		"/structs.structs.MsgStructOreRefineryComplete": types.PermHashRefine,
		"/structs.structs.MsgPlanetRaidComplete":        types.PermHashRaid,
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

func TestGuildBankMintRequiresTokenTransfer(t *testing.T) {
	actualPerm, ok := PermissionMap["/structs.structs.MsgGuildBankMint"]
	require.True(t, ok, "MsgGuildBankMint missing from PermissionMap")
	require.NotZero(t, actualPerm&types.PermGuildTokenMint)
	require.NotZero(t, actualPerm&types.PermTokenTransfer,
		"minting debits alpha from the primary address, so mint authority alone is not enough")
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
	// These cache methods hide a primary-account debit from the handler AST.
	// Keep the list explicit so adding another such method requires classifying it.
	primaryDebitHelpers := map[string]bool{
		"BankConvert": true,
		"BankMint":    true,
		"BankRedeem":  true,
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
				if primaryDebitHelpers[called] {
					readsPrimary = true
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

// throttledMessages returns every message whose throttle key names an object the
// transaction chose, which is the set that needs target authorization before the
// ante may reserve for it.
func throttledMessages() map[string]bool {
	throttled := make(map[string]bool, len(ProofMessages)+len(ThrottleKeyExtractors))
	for typeURL := range ProofMessages {
		throttled[typeURL] = true
	}
	for typeURL := range ThrottleKeyExtractors {
		throttled[typeURL] = true
	}
	return throttled
}

// TestThrottleTargetAuthCompleteness keeps ThrottleTargetAuth in step with the
// two maps that create object-global throttle keys.
//
// A missing entry does not error, it disarms: mayReserve treats an unmapped
// message as unauthorized, so that message stops reserving and its throttle
// quietly stops working. An extra entry is dead weight that suggests a throttle
// exists where none does.
func TestThrottleTargetAuthCompleteness(t *testing.T) {
	throttled := throttledMessages()
	require.NotEmpty(t, throttled, "no throttled messages found; test would be vacuous")

	for typeURL := range throttled {
		require.Contains(t, ThrottleTargetAuth, typeURL,
			"%s reserves an object-global throttle key but has no ThrottleTargetAuth entry, so it now reserves nothing at all", typeURL)
	}

	for typeURL := range ThrottleTargetAuth {
		require.True(t, throttled[typeURL],
			"%s has a ThrottleTargetAuth entry but is in neither ProofMessages nor ThrottleKeyExtractors", typeURL)
		require.True(t, KnownStructsMessages[typeURL],
			"ThrottleTargetAuth entry %s not in KnownStructsMessages", typeURL)
	}
}

// TestArch_ThrottleTargetAuthMatchesHandlers ties the permission the ante uses
// to reserve a throttle key to the one the handler actually demands.
//
// The two have to agree in one direction more than the other. Declaring a bit
// stricter than the handler's under-throttles — the legitimate actor reserves
// nothing and the object goes unthrottled for the block. Declaring one weaker
// re-opens the hole this map was added to close, by letting a signer reserve for
// an object it cannot act on. Neither shows up as an error at runtime, so the
// handler sources are the enforcement.
func TestArch_ThrottleTargetAuthMatchesHandlers(t *testing.T) {
	// Every Can*By method a throttled handler is allowed to authorize with, and
	// the permission it resolves to in x/structs/keeper/player_cache.go.
	permByCanMethod := map[string]types.Permission{
		"CanBePlayedBy":     types.PermPlay,
		"CanBuildHashedBy":  types.PermHashBuild,
		"CanMineHashedBy":   types.PermHashMine,
		"CanRefineHashedBy": types.PermHashRefine,
		"CanRaidHashedBy":   types.PermHashRaid,
	}
	// Methods taking the permission as an argument, so the bit is whatever the
	// message asks for and no fixed value can be asserted.
	dynamicCanMethods := map[string]bool{
		"CanRegisterAddressBy": true,
	}

	registry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)

	keeperDir := filepath.Join("..", "..", "x", "structs", "keeper")
	entries, err := os.ReadDir(keeperDir)
	require.NoError(t, err)

	fset := token.NewFileSet()
	var failures []string
	checked := map[string]bool{}

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
			typeURL := "/structs.structs." + msgName
			extractor, throttled := ThrottleTargetAuth[typeURL]
			if !throttled {
				continue
			}
			checked[typeURL] = true

			// A signer-scoped key authorizes the signer as themselves, which is
			// not what the handler's Can*By calls are about — MsgGuildCreate's
			// authorize the founder, who need not be the signer. Asserting the
			// mirror here would demand the wrong thing; the property that does
			// hold is checked in TestThrottleTargetAuthTargetsMatchThrottleKeys.
			if SignerScopedThrottleMessages[typeURL] {
				continue
			}

			var authMethods []string
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				called := sel.Sel.Name
				if strings.HasPrefix(called, "Can") && strings.HasSuffix(called, "By") {
					authMethods = append(authMethods, called)
				}
				return true
			})

			if len(authMethods) != 1 {
				failures = append(failures, name+": "+msgName+" authorizes with "+
					strconv.Itoa(len(authMethods))+" Can*By calls ("+strings.Join(authMethods, ", ")+
					"); ThrottleTargetAuth can only mirror exactly one")
				continue
			}
			method := authMethods[0]

			msg, err := registry.Resolve(typeURL)
			require.NoError(t, err, "could not resolve %s", typeURL)

			if dynamicCanMethods[method] {
				// The handler passes the message's own field through, so the map
				// has to read the same field rather than name a bit.
				const probe = types.Permission(1 << 9)
				field := reflect.ValueOf(msg).Elem().FieldByName("Permissions")
				require.True(t, field.IsValid() && field.Kind() == reflect.Uint64,
					"%s authorizes with %s but has no uint64 Permissions field", msgName, method)
				field.SetUint(uint64(probe))

				target, ok := extractor(msg)
				if !ok || target.Permission != probe {
					failures = append(failures, name+": "+msgName+" authorizes with "+method+
						", whose bit comes from the message, but ThrottleTargetAuth does not pass Permissions through")
				}
				continue
			}

			want, known := permByCanMethod[method]
			if !known {
				failures = append(failures, name+": "+msgName+" authorizes with "+method+
					", which this guard does not know; add it to permByCanMethod or dynamicCanMethods")
				continue
			}

			target, ok := extractor(msg)
			if !ok {
				failures = append(failures, name+": ThrottleTargetAuth entry for "+msgName+
					" does not accept its own message type")
				continue
			}
			if target.Permission != want {
				failures = append(failures, name+": "+msgName+" authorizes with "+method+
					" ("+strconv.FormatUint(uint64(want), 10)+") but ThrottleTargetAuth declares "+
					strconv.FormatUint(uint64(target.Permission), 10))
			}
		}
	}

	for typeURL := range ThrottleTargetAuth {
		require.True(t, checked[typeURL],
			"no handler source found for %s; the scan is probably broken", typeURL)
	}

	require.Empty(t, failures,
		"ThrottleTargetAuth must demand the same permission its handler does:\n  - %s",
		strings.Join(failures, "\n  - "))
}

// TestThrottleTargetAuthTargetsMatchThrottleKeys checks the other half of the
// mirror: the object the map authorizes against has to be the object the
// throttle key names, or the ante would be checking standing on one thing and
// reserving another.
func TestThrottleTargetAuthTargetsMatchThrottleKeys(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)

	// The id field each message puts in its throttle key, and the object kind
	// that id names.
	idFields := map[string]struct {
		field string
		kind  types.ObjectType
	}{
		"/structs.structs.MsgStructBuildComplete":       {"StructId", types.ObjectType_struct},
		"/structs.structs.MsgStructOreMinerComplete":    {"StructId", types.ObjectType_struct},
		"/structs.structs.MsgStructOreRefineryComplete": {"StructId", types.ObjectType_struct},
		"/structs.structs.MsgPlanetRaidComplete":        {"FleetId", types.ObjectType_fleet},
		"/structs.structs.MsgFleetMove":                 {"FleetId", types.ObjectType_fleet},
		"/structs.structs.MsgPlanetExplore":             {"PlayerId", types.ObjectType_player},
		"/structs.structs.MsgAddressRegister":           {"PlayerId", types.ObjectType_player},
	}

	// Signer-scoped messages name the creator instead of an object, so the id
	// they have to agree on is the signing address.
	for typeURL := range SignerScopedThrottleMessages {
		extractor, hasAuth := ThrottleTargetAuth[typeURL]
		require.True(t, hasAuth, "%s is signer-scoped but has no ThrottleTargetAuth entry", typeURL)

		msg, err := registry.Resolve(typeURL)
		require.NoError(t, err)

		const signer = "structs1sentinelsigneraddress"
		field := reflect.ValueOf(msg).Elem().FieldByName("Creator")
		require.True(t, field.IsValid() && field.Kind() == reflect.String,
			"%s has no string Creator field", typeURL)
		field.SetString(signer)

		target, ok := extractor(msg)
		require.True(t, ok, "ThrottleTargetAuth entry for %s rejected its own message type", typeURL)
		require.Equal(t, types.ObjectType_address, target.Kind,
			"a signer-scoped %s must authorize against the signing address, not an object", typeURL)
		require.Equal(t, signer, target.TargetId,
			"ThrottleTargetAuth for %s authorizes against something other than its signer", typeURL)

		if proofExtractor, hasProof := ProofMessages[typeURL]; hasProof {
			require.Contains(t, proofExtractor(msg), signer,
				"%s is signer-scoped but reserves a key that does not name its signer, so one attempt would throttle everyone", typeURL)
		}
		if keyExtractor, hasKey := ThrottleKeyExtractors[typeURL]; hasKey {
			require.Contains(t, keyExtractor(msg), signer,
				"%s is signer-scoped but reserves a key that does not name its signer", typeURL)
		}
	}

	for typeURL, extractor := range ThrottleTargetAuth {
		if SignerScopedThrottleMessages[typeURL] {
			continue
		}

		spec, known := idFields[typeURL]
		require.True(t, known, "%s has no id field declared in this test", typeURL)

		msg, err := registry.Resolve(typeURL)
		require.NoError(t, err)

		const sentinel = "sentinel-object-id"
		field := reflect.ValueOf(msg).Elem().FieldByName(spec.field)
		require.True(t, field.IsValid() && field.Kind() == reflect.String,
			"%s has no string %s field", typeURL, spec.field)
		field.SetString(sentinel)

		target, ok := extractor(msg)
		require.True(t, ok, "ThrottleTargetAuth entry for %s rejected its own message type", typeURL)
		require.Equal(t, sentinel, target.TargetId,
			"ThrottleTargetAuth for %s authorizes against something other than the id in its throttle key", typeURL)
		require.Equal(t, spec.kind, target.Kind,
			"ThrottleTargetAuth for %s names the wrong object kind, so the keeper resolves the wrong owner", typeURL)

		// The throttle key must be built from that same id.
		if keyExtractor, hasKey := ThrottleKeyExtractors[typeURL]; hasKey {
			require.Contains(t, keyExtractor(msg), sentinel,
				"%s reserves a key that does not name the object it authorizes against", typeURL)
		}
		if proofExtractor, hasProof := ProofMessages[typeURL]; hasProof {
			require.Equal(t, sentinel, proofExtractor(msg),
				"%s reserves a proof key that does not name the object it authorizes against", typeURL)
		}
	}
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
