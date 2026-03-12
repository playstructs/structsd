package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPermissionGuildRankSet(t *testing.T) {
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

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(owner.Creator)
	testPermissionAdd(k, ctx, addressPermissionId, types.PermAdmin)

	// Success: set guild rank permission
	resp, err := ms.PermissionGuildRankSet(wctx, &types.MsgPermissionGuildRankSet{
		Creator:     owner.Creator,
		ObjectId:    structObj.Id,
		GuildId:     guild.Id,
		Permission:  uint64(types.Permission(1)),
		HighestRank: 2,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	rank, ok := k.GetHighestGuildRankForPermission(ctx, structObj.Id, guild.Id, types.Permission(1))
	require.True(t, ok)
	require.Equal(t, uint64(2), rank)

	// Validation: empty object_id
	_, err = ms.PermissionGuildRankSet(wctx, &types.MsgPermissionGuildRankSet{
		Creator:     owner.Creator,
		ObjectId:    "",
		GuildId:     guild.Id,
		Permission:  1,
		HighestRank: 0,
	})
	require.Error(t, err)

	// Validation: empty guild_id
	_, err = ms.PermissionGuildRankSet(wctx, &types.MsgPermissionGuildRankSet{
		Creator:     owner.Creator,
		ObjectId:    structObj.Id,
		GuildId:     "",
		Permission:  1,
		HighestRank: 0,
	})
	require.Error(t, err)

	// Validation: permission 0
	_, err = ms.PermissionGuildRankSet(wctx, &types.MsgPermissionGuildRankSet{
		Creator:     owner.Creator,
		ObjectId:    structObj.Id,
		GuildId:     guild.Id,
		Permission:  0,
		HighestRank: 0,
	})
	require.Error(t, err)

	// Guild not found
	_, err = ms.PermissionGuildRankSet(wctx, &types.MsgPermissionGuildRankSet{
		Creator:     owner.Creator,
		ObjectId:    structObj.Id,
		GuildId:     "2-99999",
		Permission:  1,
		HighestRank: 0,
	})
	require.Error(t, err)
}
