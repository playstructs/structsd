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
	guild1 := "2-1"
	guild2 := "2-2"

	keeper.SetHighestGuildRankPermission(ctx, objectId, guild1, types.Permission(1), 3)
	keeper.SetHighestGuildRankPermission(ctx, objectId, guild2, types.Permission(2), 1)

	// Valid request
	resp, err := keeper.GuildRankPermissionByObject(wctx, &types.QueryGuildRankPermissionByObjectRequest{
		ObjectId: objectId,
	})
	require.NoError(t, err)
	require.Len(t, resp.GuildRankPermissionRecords, 2)

	// With pagination
	resp2, err := keeper.GuildRankPermissionByObject(wctx, &types.QueryGuildRankPermissionByObjectRequest{
		ObjectId: objectId,
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp2.GuildRankPermissionRecords, 1)
	require.NotNil(t, resp2.Pagination.NextKey)

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
	guildId := "2-3"

	keeper.SetHighestGuildRankPermission(ctx, objectId, guildId, types.Permission(1), 2)
	keeper.SetHighestGuildRankPermission(ctx, objectId, guildId, types.Permission(4), 0)

	resp, err := keeper.GuildRankPermissionByObjectAndGuild(wctx, &types.QueryGuildRankPermissionByObjectAndGuildRequest{
		ObjectId: objectId,
		GuildId:  guildId,
	})
	require.NoError(t, err)
	require.Len(t, resp.GuildRankPermissionRecords, 2)

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
