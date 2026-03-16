package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructDefenseSet(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, ctx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))

	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

	structType := types.StructType{
		Id:                 1,
		Type:               types.CommandStruct,
		Category:           types.ObjectType_player,
		DefendChangeCharge: 10,
	}
	k.SetStructType(ctx, structType)

	t.Run("defender struct not found", func(t *testing.T) {
		protectedStruct := types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		}
		protectedStruct = testAppendStruct(k, ctx, protectedStruct)

		_, err := ms.StructDefenseSet(wctx, &types.MsgStructDefenseSet{
			Creator:           player.Creator,
			DefenderStructId:  "invalid-struct",
			ProtectedStructId: protectedStruct.Id,
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructDefenseSet(wctx, &types.MsgStructDefenseSet{
			Creator:           "cosmos1noperms",
			DefenderStructId:  "5-1",
			ProtectedStructId: "5-2",
		})
		require.Error(t, err)
	})
}
