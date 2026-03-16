package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructStealthDeactivate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, sdkCtx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(100000))

	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(0))

	structType := types.StructType{
		Id:                    1,
		Type:                  types.CommandStruct,
		Category:              types.ObjectType_player,
		UnitDefenses:          types.TechUnitDefenses_stealthMode,
		StealthActivateCharge: 10,
	}
	k.SetStructType(sdkCtx, structType)

	structObj := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	structObj = testAppendStruct(k, sdkCtx, structObj)

	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateOnline))

	t.Run("valid stealth deactivation", func(t *testing.T) {
		testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateHidden))
		k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(0))
		resp, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  player.Creator,
			StructId: "invalid-struct",
		})
		require.Error(t, err)
	})

	t.Run("not in stealth", func(t *testing.T) {
		testSetStructAttributeFlagRemove(k, sdkCtx, statusAttrId, uint64(types.StructStateHidden))
		_, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.Error(t, err)
	})

	t.Run("no stealth system", func(t *testing.T) {
		testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateHidden))
		noStealthType := types.StructType{
			Id:       2,
			Type:     types.CommandStruct,
			Category: types.ObjectType_player,
		}
		k.SetStructType(sdkCtx, noStealthType)
		structObj.Type = noStealthType.Id
		k.SetStruct(sdkCtx, structObj)

		_, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.Error(t, err)

		structObj.Type = structType.Id
		k.SetStruct(sdkCtx, structObj)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  "cosmos1noperms",
			StructId: structObj.Id,
		})
		require.Error(t, err)
	})
}
