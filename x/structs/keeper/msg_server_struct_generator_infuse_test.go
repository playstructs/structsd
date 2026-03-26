package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructGeneratorInfuse(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, ctx, player)

	planet := testAppendPlanet(k, ctx, types.Planet{Creator: player.Creator, Owner: player.Id})

	structType := types.StructType{
		Id:              1,
		Type:            types.CommandStruct,
		Category:        types.ObjectType_player,
		PowerGeneration: 1,
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

	playerAcc, _ := sdk.AccAddressFromBech32(player.Creator)
	coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1000)))
	k.BankKeeper().MintCoins(ctx, types.ModuleName, coins)
	k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, coins)

	t.Run("valid generator infuse", func(t *testing.T) {
		resp, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     structObj.Id,
			InfuseAmount: "1000ualpha",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct offline", func(t *testing.T) {
		testSetStructAttributeFlagRemove(k, ctx, statusAttrId, uint64(types.StructStateOnline))

		resp, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     structObj.Id,
			InfuseAmount: "1000ualpha",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "is offline")
		_ = resp

		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))
	})

	t.Run("no power generation", func(t *testing.T) {
		noGenType := types.StructType{
			Id:              2,
			Type:            types.CommandStruct,
			Category:        types.ObjectType_player,
			PowerGeneration: types.TechPowerGeneration_noPowerGeneration,
		}
		k.SetStructType(ctx, noGenType)
		structObj.Type = noGenType.Id
		k.SetStruct(ctx, structObj)

		resp, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      player.Creator,
			StructId:     structObj.Id,
			InfuseAmount: "1000ualpha",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "has no generation system")
		_ = resp

		structObj.Type = structType.Id
		k.SetStruct(ctx, structObj)
	})

	t.Run("no permissions", func(t *testing.T) {
		resp, err := ms.StructGeneratorInfuse(wctx, &types.MsgStructGeneratorInfuse{
			Creator:      "cosmos1noperms",
			StructId:     structObj.Id,
			InfuseAmount: "1000ualpha",
		})
		require.Error(t, err)
		_ = resp
	})
}
