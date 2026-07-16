package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// TestMsgStructTrash verifies the StructTrash message. Trash mirrors
// StructBuildCancel but destroys any non-destroyed struct (built or still
// building) after a play-permission check, gates on the owner holding at least
// the struct type's BuildCharge, and consumes that charge on success.
//
// GetCharge() == BlockHeight - lastAction, so the block height is pinned and
// lastAction is set per player to make charge deterministic.
func TestMsgStructTrash(t *testing.T) {
	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx).WithBlockHeight(100)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, ctx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))

	// Ample charge for the primary player (charge = 100 - 0 = 100 >= 10).
	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

	structType := types.StructType{
		Id:          1,
		Type:        types.CommandStruct,
		Category:    types.ObjectType_player,
		BuildCharge: 10,
		BuildDraw:   100,
	}
	k.SetStructType(ctx, structType)

	t.Run("valid trash of a building struct", func(t *testing.T) {
		// Reset charge: a successful trash discharges the player (sets lastAction
		// to the block height), so each success case restores full charge first.
		k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

		structObj := testAppendStruct(k, ctx, types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		})
		blockStartAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structObj.Id)
		k.SetStructAttribute(ctx, blockStartAttrId, uint64(1))

		resp, err := ms.StructTrash(ctx, &types.MsgStructTrash{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
		require.NotZero(t, k.GetStructAttribute(ctx, statusAttrId)&uint64(types.StructStateDestroyed),
			"struct should be flagged destroyed after trash")
	})

	t.Run("valid trash of a built struct", func(t *testing.T) {
		k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

		builtStruct := testAppendStruct(k, ctx, types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		})
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, builtStruct.Id)
		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))

		resp, err := ms.StructTrash(ctx, &types.MsgStructTrash{
			Creator:  player.Creator,
			StructId: builtStruct.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotZero(t, k.GetStructAttribute(ctx, statusAttrId)&uint64(types.StructStateDestroyed),
			"built struct should be flagged destroyed after trash")
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructTrash(ctx, &types.MsgStructTrash{
			Creator:  player.Creator,
			StructId: "invalid-struct",
		})
		require.Error(t, err)
	})

	t.Run("struct already destroyed", func(t *testing.T) {
		destroyedStruct := testAppendStruct(k, ctx, types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		})
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, destroyedStruct.Id)
		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateDestroyed))

		_, err := ms.StructTrash(ctx, &types.MsgStructTrash{
			Creator:  player.Creator,
			StructId: destroyedStruct.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "destroyed")
	})

	t.Run("insufficient charge", func(t *testing.T) {
		// A dedicated owner whose lastAction equals the current block height has
		// zero charge (100 - 100 = 0 < BuildCharge 10).
		brokePlayer := testAppendPlayer(k, ctx, types.Player{
			Creator:        "cosmos1broke",
			PrimaryAddress: "cosmos1broke",
		})
		k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, brokePlayer.Id), uint64(100000))
		k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, brokePlayer.Id), uint64(100))

		structObj := testAppendStruct(k, ctx, types.Struct{
			Creator: brokePlayer.Creator,
			Owner:   brokePlayer.Id,
			Type:    structType.Id,
		})

		_, err := ms.StructTrash(ctx, &types.MsgStructTrash{
			Creator:  brokePlayer.Creator,
			StructId: structObj.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "charge")
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructTrash(ctx, &types.MsgStructTrash{
			Creator:  "cosmos1noperms",
			StructId: "5-1",
		})
		require.Error(t, err)
	})
}
