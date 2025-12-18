package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/keeper"
	kpr "structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createNGuildMembershipApplication(keeper keeper.Keeper, ctx sdk.Context, n int) []types.GuildMembershipApplication {
	items := make([]types.GuildMembershipApplication, n)
	for i := range items {
		items[i].GuildId = "guild" + string(rune(i))
		items[i].PlayerId = "player" + string(rune(i))
		items[i].JoinType = types.GuildJoinType_request
		items[i].RegistrationStatus = types.RegistrationStatus_proposed
		items[i].Proposer = "proposer" + string(rune(i))
		items[i].SubstationId = "substation" + string(rune(i))
		keeper.SetGuildMembershipApplication(ctx, items[i])
	}
	return items
}

func TestGuildMembershipApplicationGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuildMembershipApplication(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetGuildMembershipApplication(ctx, item.GuildId, item.PlayerId)
		require.True(t, found)
		require.Equal(t, item.GuildId, rst.GuildId)
		require.Equal(t, item.PlayerId, rst.PlayerId)
		require.Equal(t, item.JoinType, rst.JoinType)
		require.Equal(t, item.RegistrationStatus, rst.RegistrationStatus)
		require.Equal(t, item.Proposer, rst.Proposer)
		require.Equal(t, item.SubstationId, rst.SubstationId)
	}
}

func TestGuildMembershipApplicationRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNGuildMembershipApplication(keeper, ctx, 10)
	for _, item := range items {
		keeper.ClearGuildMembershipApplication(ctx, item.GuildId, item.PlayerId)
		_, found := keeper.GetGuildMembershipApplication(ctx, item.GuildId, item.PlayerId)
		require.False(t, found)
	}
}

func TestGuildMembershipApplicationSet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	item := types.GuildMembershipApplication{
		GuildId:            "guild1",
		PlayerId:           "player1",
		JoinType:           types.GuildJoinType_request,
		RegistrationStatus: types.RegistrationStatus_proposed,
		Proposer:           "proposer1",
		SubstationId:       "substation1",
	}
	keeper.SetGuildMembershipApplication(ctx, item)
	rst, found := keeper.GetGuildMembershipApplication(ctx, item.GuildId, item.PlayerId)
	require.True(t, found)
	require.Equal(t, item.GuildId, rst.GuildId)
	require.Equal(t, item.PlayerId, rst.PlayerId)
	require.Equal(t, item.JoinType, rst.JoinType)
	require.Equal(t, item.RegistrationStatus, rst.RegistrationStatus)
	require.Equal(t, item.Proposer, rst.Proposer)
	require.Equal(t, item.SubstationId, rst.SubstationId)
}

func TestGuildMembershipApplicationID(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	guildId := "guild1"
	playerId := "player1"
	expectedId := playerId + "@" + guildId

	// Create and set an application
	application := types.GuildMembershipApplication{
		GuildId:            guildId,
		PlayerId:           playerId,
		JoinType:           types.GuildJoinType_request,
		RegistrationStatus: types.RegistrationStatus_proposed,
		Proposer:           "proposer1",
		SubstationId:       "substation1",
	}
	keeper.SetGuildMembershipApplication(ctx, application)

	// Verify application can be retrieved using the same ID format
	got, found := keeper.GetGuildMembershipApplication(ctx, guildId, playerId)
	require.True(t, found)
	require.Equal(t, kpr.GetGuildMembershipApplicationID(guildId, playerId), expectedId)
	require.Equal(t, application.GuildId, got.GuildId)
	require.Equal(t, application.PlayerId, got.PlayerId)
}
