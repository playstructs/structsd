package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestGuildRankPermissionByObjectQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	objectId := "1-5"
	guild1 := "0-1"
	guild2 := "0-2"

	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guild1, types.PermPlay, 3)
	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guild2, types.PermAdmin, 1)

	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guild1, types.PermDelete, 7)
	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guild2, types.PermUpdate, 2)
	keeper.SetGuildRankPermissionStoreOnly(ctx, "1-50", guild1, types.PermAll, 9)
	expected := [][]*types.GuildRankPermissionRecord{
		{
			{ObjectId: objectId, GuildId: guild1, Permissions: uint64(types.PermPlay), Rank: 3},
			{ObjectId: objectId, GuildId: guild1, Permissions: uint64(types.PermDelete), Rank: 7},
		},
		{
			{ObjectId: objectId, GuildId: guild2, Permissions: uint64(types.PermAdmin), Rank: 1},
			{ObjectId: objectId, GuildId: guild2, Permissions: uint64(types.PermUpdate), Rank: 2},
		},
	}
	for _, reverse := range []bool{false, true} {
		var next []byte
		for page := 0; page < 2; page++ {
			index := page
			if reverse {
				index = 1 - page
			}
			req := &types.QueryGuildRankPermissionByObjectRequest{ObjectId: objectId,
				Pagination: &query.PageRequest{Key: next, Limit: 1, Reverse: reverse, CountTotal: true}}
			resp, err := keeper.GuildRankPermissionByObject(wctx, req)
			require.NoError(t, err)
			require.Equal(t, expected[index], resp.GuildRankPermissionRecords, "a page contains every nonzero bit in its selected guild register")
			require.Zero(t, resp.Pagination.Total)
			next = resp.Pagination.NextKey
			offsetResp, err := keeper.GuildRankPermissionByObject(wctx, &types.QueryGuildRankPermissionByObjectRequest{
				ObjectId: objectId, Pagination: &query.PageRequest{Offset: uint64(page), Limit: 1, Reverse: reverse}})
			require.NoError(t, err)
			require.Equal(t, expected[index], offsetResp.GuildRankPermissionRecords)
		}
		require.Empty(t, next)
	}
	resp, err := keeper.GuildRankPermissionByObject(wctx, &types.QueryGuildRankPermissionByObjectRequest{ObjectId: objectId})
	require.NoError(t, err)
	require.Equal(t, append(expected[0], expected[1]...), resp.GuildRankPermissionRecords)

	// Invalid: nil request
	_, err = keeper.GuildRankPermissionByObject(wctx, nil)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	// Invalid: empty object_id
	_, err = keeper.GuildRankPermissionByObject(wctx, &types.QueryGuildRankPermissionByObjectRequest{
		ObjectId: "",
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	// Empty result for unknown object
	respEmpty, err := keeper.GuildRankPermissionByObject(wctx, &types.QueryGuildRankPermissionByObjectRequest{
		ObjectId: "1-99999",
	})
	require.NoError(t, err)
	require.Empty(t, respEmpty.GuildRankPermissionRecords)
}

func TestGuildRankPermissionByObjectAndGuildQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	objectId := "1-6"
	guildId := "0-3"

	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guildId, types.PermPlay, 2)
	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guildId, types.PermUpdate, 5)

	resp, err := keeper.GuildRankPermissionByObjectAndGuild(wctx, &types.QueryGuildRankPermissionByObjectAndGuildRequest{
		ObjectId: objectId,
		GuildId:  guildId,
	})
	require.NoError(t, err)
	require.Equal(t, []*types.GuildRankPermissionRecord{
		{ObjectId: objectId, GuildId: guildId, Permissions: uint64(types.PermPlay), Rank: 2},
		{ObjectId: objectId, GuildId: guildId, Permissions: uint64(types.PermUpdate), Rank: 5},
	}, resp.GuildRankPermissionRecords)

	// Invalid: empty object_id
	_, err = keeper.GuildRankPermissionByObjectAndGuild(wctx, &types.QueryGuildRankPermissionByObjectAndGuildRequest{
		ObjectId: "",
		GuildId:  guildId,
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	// Invalid: empty guild_id
	_, err = keeper.GuildRankPermissionByObjectAndGuild(wctx, &types.QueryGuildRankPermissionByObjectAndGuildRequest{
		ObjectId: objectId,
		GuildId:  "",
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
