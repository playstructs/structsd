package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPermissionGuildRankRevoke(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	ownerAcc := sdk.AccAddress("owner123456789012345678901234567890")
	owner := types.Player{
		Creator:        ownerAcc.String(),
		PrimaryAddress: ownerAcc.String(),
	}
	owner = testAppendPlayer(k, ctx, owner)

	structObj := types.Struct{
		Creator: owner.Creator,
		Owner:   owner.Id,
		Type:    1,
	}
	structObj = testAppendStruct(k, ctx, structObj)

	validatorAddress := sdk.ValAddress(ownerAcc.Bytes())
	reactor := types.Reactor{RawAddress: validatorAddress.Bytes()}
	reactor = k.AppendReactor(ctx, reactor)

	guild := k.AppendGuild(ctx, "test-endpoint", "", reactor, owner)

	k.SetGuildRankPermission(ctx, structObj.Id, guild.Id, types.Permission(1), 2)
	rank, ok := k.GetGuildRankForPermission(ctx, structObj.Id, guild.Id, types.Permission(1))
	require.True(t, ok)
	require.Equal(t, uint64(2), rank)

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(owner.Creator)
	testPermissionAdd(k, ctx, addressPermissionId, types.PermAdmin)

	resp, err := ms.PermissionGuildRankRevoke(wctx, &types.MsgPermissionGuildRankRevoke{
		Creator:    owner.Creator,
		ObjectId:   structObj.Id,
		GuildId:    guild.Id,
		Permission: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	_, ok = k.GetGuildRankForPermission(ctx, structObj.Id, guild.Id, types.Permission(1))
	require.False(t, ok)

	// Validation: empty object_id
	_, err = ms.PermissionGuildRankRevoke(wctx, &types.MsgPermissionGuildRankRevoke{
		Creator:   owner.Creator,
		ObjectId:  "",
		GuildId:   guild.Id,
		Permission: 1,
	})
	require.Error(t, err)

	// Validation: permission 0
	_, err = ms.PermissionGuildRankRevoke(wctx, &types.MsgPermissionGuildRankRevoke{
		Creator:   owner.Creator,
		ObjectId:  structObj.Id,
		GuildId:   guild.Id,
		Permission: 0,
	})
	require.Error(t, err)
}
