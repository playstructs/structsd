package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructBuildInitiate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, sdkCtx, player)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{Owner: player.Id})
	_ = fleet

	cc := k.NewCurrentContext(sdkCtx)
	playerCache, err := cc.GetPlayer(player.Id)
	require.NoError(t, err)
	playerCache.SetFleetId(fleet.Id)
	playerCache.Commit()

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(100000))

	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(0))

	structType := types.StructType{
		Id:            1,
		Type:          types.CommandStruct,
		Category:      types.ObjectType_fleet,
		BuildCharge:   10,
		BuildDraw:     100,
		PossibleAmbit: types.Ambit_flag[types.Ambit_space],
	}
	k.SetStructType(sdkCtx, structType)

	t.Run("valid struct build initiation", func(t *testing.T) {
		resp, err := ms.StructBuildInitiate(wctx, &types.MsgStructBuildInitiate{
			Creator:        player.Creator,
			PlayerId:       player.Id,
			StructTypeId:   structType.Id,
			OperatingAmbit: types.Ambit_space,
			Slot:           0,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct type not found", func(t *testing.T) {
		_, err := ms.StructBuildInitiate(wctx, &types.MsgStructBuildInitiate{
			Creator:        player.Creator,
			PlayerId:       player.Id,
			StructTypeId:   999,
			OperatingAmbit: types.Ambit_space,
			Slot:           1,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructBuildInitiate(wctx, &types.MsgStructBuildInitiate{
			Creator:        "cosmos1noperms",
			PlayerId:       player.Id,
			StructTypeId:   structType.Id,
			OperatingAmbit: types.Ambit_space,
			Slot:           1,
		})
		require.Error(t, err)
	})

	t.Run("insufficient charge", func(t *testing.T) {
		k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(sdkCtx.BlockHeight()))
		_, err := ms.StructBuildInitiate(wctx, &types.MsgStructBuildInitiate{
			Creator:        player.Creator,
			PlayerId:       player.Id,
			StructTypeId:   structType.Id,
			OperatingAmbit: types.Ambit_space,
			Slot:           1,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "required charge")
	})
}
