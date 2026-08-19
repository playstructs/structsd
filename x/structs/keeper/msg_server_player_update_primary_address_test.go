package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPlayerUpdatePrimaryAddress(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	newPrimaryAcc := sdk.AccAddress("newprimary123456789012345678901234567890")
	newPrimaryAddress := newPrimaryAcc.String()
	k.SetPlayerIndexForAddress(ctx, newPrimaryAddress, player.Index)

	newPrimaryPermissionId := keeperlib.GetAddressPermissionIDBytes(newPrimaryAddress)
	testPermissionAdd(k, ctx, newPrimaryPermissionId, types.PermAll)

	t.Run("valid primary address update", func(t *testing.T) {
		resp, err := ms.PlayerUpdatePrimaryAddress(wctx, &types.MsgPlayerUpdatePrimaryAddress{
			Creator:        player.Creator,
			PrimaryAddress: newPrimaryAddress,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPlayer, found := k.GetPlayer(ctx, player.Id)
		require.True(t, found)
		require.Equal(t, newPrimaryAddress, updatedPlayer.PrimaryAddress)
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregistered := sdk.AccAddress("unregistered12345678901234567890123").String()
		resp, err := ms.PlayerUpdatePrimaryAddress(wctx, &types.MsgPlayerUpdatePrimaryAddress{
			Creator:        unregistered,
			PrimaryAddress: newPrimaryAddress,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not associated")
		_ = resp
	})
}

func TestMsgPlayerUpdatePrimaryAddress_ReducedCallerRejected(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("reducedcaller123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	callerPermissionId := keeperlib.GetAddressPermissionIDBytes(player.PrimaryAddress)
	k.SetPermissionsByBytes(ctx, callerPermissionId, types.PermAll&^types.PermDelete)

	newPrimaryAcc := sdk.AccAddress("reducednewprim123456789012345678901234567890")
	newPrimaryAddress := newPrimaryAcc.String()
	k.SetPlayerIndexForAddress(ctx, newPrimaryAddress, player.Index)
	testPermissionAdd(k, ctx, keeperlib.GetAddressPermissionIDBytes(newPrimaryAddress), types.PermPlay)

	resp, err := ms.PlayerUpdatePrimaryAddress(wctx, &types.MsgPlayerUpdatePrimaryAddress{
		Creator:        player.Creator,
		PrimaryAddress: newPrimaryAddress,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission")
	_ = resp

	unchanged, found := k.GetPlayer(ctx, player.Id)
	require.True(t, found)
	require.Equal(t, player.PrimaryAddress, unchanged.PrimaryAddress,
		"primary address must not change when the caller lacks PermAll")

	newPerms := k.GetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(newPrimaryAddress))
	require.Equal(t, types.PermPlay, newPerms,
		"incoming address must not be upgraded when the update is rejected")
}

func TestMsgPlayerUpdatePrimaryAddress_GrantsFullPermissions(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("fullgrantcaller123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	newPrimaryAcc := sdk.AccAddress("fullgrantnewprim123456789012345678901234567890")
	newPrimaryAddress := newPrimaryAcc.String()
	k.SetPlayerIndexForAddress(ctx, newPrimaryAddress, player.Index)

	newPrimaryPermissionId := keeperlib.GetAddressPermissionIDBytes(newPrimaryAddress)
	k.SetPermissionsByBytes(ctx, newPrimaryPermissionId, types.PermPlay)

	resp, err := ms.PlayerUpdatePrimaryAddress(wctx, &types.MsgPlayerUpdatePrimaryAddress{
		Creator:        player.Creator,
		PrimaryAddress: newPrimaryAddress,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	updatedPlayer, found := k.GetPlayer(ctx, player.Id)
	require.True(t, found)
	require.Equal(t, newPrimaryAddress, updatedPlayer.PrimaryAddress)

	newPerms := k.GetPermissionsByBytes(ctx, newPrimaryPermissionId)
	require.Equal(t, types.PermAll, newPerms,
		"SetPrimaryAddress must upgrade the incoming address to PermAll")
}
