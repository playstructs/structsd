package ante

import (
	"reflect"
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
