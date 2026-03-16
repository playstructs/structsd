package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructBuildComplete(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, ctx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))

	structType := types.StructType{
		Id:              1,
		Type:            types.CommandStruct,
		Category:        types.ObjectType_player,
		BuildDraw:       100,
		PassiveDraw:     50,
		BuildDifficulty: 1,
	}
	k.SetStructType(ctx, structType)

	structObj := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	structObj = testAppendStruct(k, ctx, structObj)

	blockStartAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structObj.Id)
	k.SetStructAttribute(ctx, blockStartAttrId, uint64(1))

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructBuildComplete(wctx, &types.MsgStructBuildComplete{
			Creator:  player.Creator,
			StructId: "invalid-struct",
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})

	t.Run("struct already built", func(t *testing.T) {
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))

		_, err := ms.StructBuildComplete(wctx, &types.MsgStructBuildComplete{
			Creator:  player.Creator,
			StructId: structObj.Id,
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "built")
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructBuildComplete(wctx, &types.MsgStructBuildComplete{
			Creator:  "cosmos1noperms",
			StructId: structObj.Id,
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})
}
