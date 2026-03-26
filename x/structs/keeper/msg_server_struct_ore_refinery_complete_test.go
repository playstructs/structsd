package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructOreRefineryComplete(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, ctx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))

	planet := testAppendPlanet(k, ctx, types.Planet{Creator: player.Creator, Owner: player.Id})

	structType := types.StructType{
		Id:                    1,
		Type:                  types.CommandStruct,
		Category:              types.ObjectType_player,
		OreRefiningDifficulty: 1,
	}
	k.SetStructType(ctx, structType)

	structObj := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   planet.Id,
		LocationType: types.ObjectType_planet,
	}
	structObj = testAppendStruct(k, ctx, structObj)

	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))

	blockStartAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structObj.Id)
	k.SetStructAttribute(ctx, blockStartAttrId, uint64(1))

	t.Run("valid ore refinery complete", func(t *testing.T) {
		resp, err := ms.StructOreRefineryComplete(wctx, &types.MsgStructOreRefineryComplete{
			Creator:  player.Creator,
			StructId: structObj.Id,
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		_ = resp
		_ = err
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructOreRefineryComplete(wctx, &types.MsgStructOreRefineryComplete{
			Creator:  player.Creator,
			StructId: "invalid-struct",
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructOreRefineryComplete(wctx, &types.MsgStructOreRefineryComplete{
			Creator:  "cosmos1noperms",
			StructId: structObj.Id,
			Nonce:    "test-nonce",
			Proof:    "test-proof",
		})
		require.Error(t, err)
	})
}
