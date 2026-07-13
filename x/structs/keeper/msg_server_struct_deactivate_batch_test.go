package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructDeactivateBatch(t *testing.T) {
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
		Id:          1,
		Type:        types.CommandStruct,
		Category:    types.ObjectType_player,
		PassiveDraw: 50,
	}
	k.SetStructType(ctx, structType)

	// newOnlineStruct creates a built+online struct owned by owner.
	newOnlineStruct := func(owner types.Player) types.Struct {
		structObj := types.Struct{
			Creator: owner.Creator,
			Owner:   owner.Id,
			Type:    structType.Id,
		}
		structObj = testAppendStruct(k, ctx, structObj)
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))
		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))
		return structObj
	}

	isOnline := func(structId string) bool {
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId)
		return k.GetStructAttribute(ctx, statusAttrId)&uint64(types.StructStateOnline) != 0
	}

	t.Run("valid batch deactivation", func(t *testing.T) {
		s1 := newOnlineStruct(player)
		s2 := newOnlineStruct(player)

		resp, err := ms.StructDeactivateBatch(wctx, &types.MsgStructDeactivateBatch{
			Creator:  player.Creator,
			StructId: []string{s1.Id, s2.Id},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Structs, 2)
		require.False(t, isOnline(s1.Id))
		require.False(t, isOnline(s2.Id))
	})

	t.Run("empty batch rejected", func(t *testing.T) {
		_, err := ms.StructDeactivateBatch(wctx, &types.MsgStructDeactivateBatch{
			Creator:  player.Creator,
			StructId: []string{},
		})
		require.Error(t, err)
	})

	t.Run("over limit rejected", func(t *testing.T) {
		ids := make([]string, types.MaxStructDeactivateBatchSize+1)
		for i := range ids {
			ids[i] = fmt.Sprintf("%d-%d", types.ObjectType_struct, i)
		}
		_, err := ms.StructDeactivateBatch(wctx, &types.MsgStructDeactivateBatch{
			Creator:  player.Creator,
			StructId: ids,
		})
		require.Error(t, err)
	})

	t.Run("duplicate id rejected", func(t *testing.T) {
		s1 := newOnlineStruct(player)

		_, err := ms.StructDeactivateBatch(wctx, &types.MsgStructDeactivateBatch{
			Creator:  player.Creator,
			StructId: []string{s1.Id, s1.Id},
		})
		require.Error(t, err)
		require.True(t, isOnline(s1.Id), "duplicate rejection must not deactivate the struct")
	})

	t.Run("atomic failure leaves structs online", func(t *testing.T) {
		s1 := newOnlineStruct(player)
		s2 := newOnlineStruct(player)

		// s2 is already offline -> whole batch must fail, s1 stays online
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, s2.Id)
		k.SetStructAttribute(ctx, statusAttrId, uint64(types.StructStateBuilt))

		_, err := ms.StructDeactivateBatch(wctx, &types.MsgStructDeactivateBatch{
			Creator:  player.Creator,
			StructId: []string{s1.Id, s2.Id},
		})
		require.Error(t, err)
		require.True(t, isOnline(s1.Id), "atomic batch must not partially deactivate")
	})

	t.Run("non-existent id rejected", func(t *testing.T) {
		s1 := newOnlineStruct(player)

		_, err := ms.StructDeactivateBatch(wctx, &types.MsgStructDeactivateBatch{
			Creator:  player.Creator,
			StructId: []string{s1.Id, "invalid-struct"},
		})
		require.Error(t, err)
		require.True(t, isOnline(s1.Id))
	})

	t.Run("batch succeeds when owner is offline", func(t *testing.T) {
		offlinePlayer := types.Player{
			Creator:        "cosmos1offline",
			PrimaryAddress: "cosmos1offline",
		}
		offlinePlayer = testAppendPlayer(k, ctx, offlinePlayer)

		offlineCapacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, offlinePlayer.Id)
		k.SetGridAttribute(ctx, offlineCapacityAttrId, uint64(1000))
		structsLoadAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, offlinePlayer.Id)
		k.SetGridAttribute(ctx, structsLoadAttrId, uint64(2000))

		s1 := newOnlineStruct(offlinePlayer)

		resp, err := ms.StructDeactivateBatch(wctx, &types.MsgStructDeactivateBatch{
			Creator:  offlinePlayer.Creator,
			StructId: []string{s1.Id},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, isOnline(s1.Id))
	})
}
