package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructAttack(t *testing.T) {
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
		Id:                   1,
		Type:                 types.CommandStruct,
		Category:             types.ObjectType_player,
		PrimaryWeapon:        1,
		PrimaryWeaponCharge:  10,
		PrimaryWeaponTargets: 1,
	}
	k.SetStructType(ctx, structType)

	attackerStruct := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   "planet1",
		LocationType: types.ObjectType_planet,
	}
	attackerStruct = testAppendStruct(k, ctx, attackerStruct)

	targetStruct := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   "planet1",
		LocationType: types.ObjectType_planet,
	}
	targetStruct = testAppendStruct(k, ctx, targetStruct)

	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, attackerStruct.Id)
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))

	targetStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, targetStruct.Id)
	testSetStructAttributeFlagAdd(k, ctx, targetStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, ctx, targetStatusAttrId, uint64(types.StructStateOnline))

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           player.Creator,
			OperatingStructId: "invalid-struct",
			WeaponSystem:      "beam",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           "cosmos1noperms",
			OperatingStructId: attackerStruct.Id,
			WeaponSystem:      "beam",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.Error(t, err)
	})
}
