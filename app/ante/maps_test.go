package ante

import (
	"testing"

	"github.com/stretchr/testify/require"

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
