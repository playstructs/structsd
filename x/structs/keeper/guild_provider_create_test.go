package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildProviderCreateFollowsGuildOwnership(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	allocation, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator, SourceObjectId: f.guild.Id, Controller: f.guild.Id, AllocationType: types.AllocationType_dynamic, Power: 100})
	require.NoError(t, err)
	station, err := f.ms.SubstationCreate(f.ctx, &types.MsgSubstationCreate{Creator: f.buyer.Creator, Owner: f.guild.Id, AllocationId: allocation.AllocationId})
	require.NoError(t, err)
	msg := &types.MsgProviderCreate{
		Creator: f.buyer.Creator, OwnerId: f.guild.Id, SubstationId: station.SubstationId,
		Rate: sdk.NewInt64Coin("ualpha", 1), AccessPolicy: types.ProviderAccessPolicy_openMarket,
		CapacityMinimum: 1, CapacityMaximum: 100, DurationMinimum: 1, DurationMaximum: 100,
		ProviderCancellationPenalty: math.LegacyZeroDec(), ConsumerCancellationPenalty: math.LegacyZeroDec(),
	}
	for _, owner := range []string{"0-999999", "0-01", "6-1"} {
		msg.OwnerId = owner
		_, err = f.ms.ProviderCreate(f.ctx, msg)
		require.Error(t, err)
	}
	msg.OwnerId = f.guild.Id
	msg.Creator = f.member.Creator
	_, err = f.ms.ProviderCreate(f.ctx, msg)
	require.Error(t, err)
	// Source access alone is not authority to create a guild's offering.
	f.k.SetGuildRankPermissionStoreOnly(f.ctx, station.SubstationId, f.guild.Id, types.PermSourceAllocation, f.member.GuildRank)
	_, err = f.ms.ProviderCreate(f.ctx, msg)
	require.Error(t, err)
	f.k.SetGuildRankPermissionStoreOnly(f.ctx, f.guild.Id, f.guild.Id, types.PermSourceAllocation, f.member.GuildRank)
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermSourceAllocation)
	_, err = f.ms.ProviderCreate(f.ctx, msg)
	require.Error(t, err)
	testPermissionAdd(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermSourceAllocation)
	_, err = f.ms.ProviderCreate(f.ctx, msg)
	require.NoError(t, err)
	providers := f.k.GetAllProvider(f.ctx)
	require.Len(t, providers, 1)
	provider := providers[0]
	require.Equal(t, f.guild.Id, provider.Owner)
	cc := f.k.NewCurrentContext(f.ctx)
	for _, id := range []string{f.member.Id, f.buyer.Id, f.guild.Id} {
		require.Equal(t, types.Permissionless, cc.GetPermissions(keeperlib.GetObjectPermissionIDBytes(provider.Id, id)))
	}
	_, err = f.ms.GuildUpdateOwnerId(f.ctx, &types.MsgGuildUpdateOwnerId{Creator: f.buyer.Creator, GuildId: f.guild.Id, Owner: f.seller.Id})
	require.NoError(t, err)
	for _, address := range []string{f.member.Creator, f.buyer.Creator} {
		_, err = f.ms.ProviderDelete(f.ctx, &types.MsgProviderDelete{Creator: address, ProviderId: provider.Id})
		require.Error(t, err)
	}
	_, err = f.ms.ProviderDelete(f.ctx, &types.MsgProviderDelete{Creator: f.seller.Creator, ProviderId: provider.Id})
	require.NoError(t, err)
}

func TestGuildProviderRankAuthorityEndsOnKick(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	allocation, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator, SourceObjectId: f.guild.Id, Controller: f.guild.Id, AllocationType: types.AllocationType_dynamic, Power: 100})
	require.NoError(t, err)
	station, err := f.ms.SubstationCreate(f.ctx, &types.MsgSubstationCreate{Creator: f.buyer.Creator, Owner: f.guild.Id, AllocationId: allocation.AllocationId})
	require.NoError(t, err)
	_, err = f.ms.ProviderCreate(f.ctx, &types.MsgProviderCreate{
		Creator: f.buyer.Creator, OwnerId: f.guild.Id, SubstationId: station.SubstationId,
		Rate: sdk.NewInt64Coin("ualpha", 1), AccessPolicy: types.ProviderAccessPolicy_openMarket,
		CapacityMinimum: 1, CapacityMaximum: 100, DurationMinimum: 1, DurationMaximum: 100,
		ProviderCancellationPenalty: math.LegacyZeroDec(), ConsumerCancellationPenalty: math.LegacyZeroDec(),
	})
	require.NoError(t, err)
	providers := f.k.GetAllProvider(f.ctx)
	require.Len(t, providers, 1)
	providerID := providers[0].Id
	_, err = f.ms.PermissionGuildRankSet(f.ctx, &types.MsgPermissionGuildRankSet{
		Creator: f.buyer.Creator, ObjectId: providerID, GuildId: f.guild.Id,
		Permission: uint64(types.PermUpdate), Rank: f.member.GuildRank,
	})
	require.NoError(t, err)
	_, err = f.ms.ProviderUpdateAccessPolicy(f.ctx, &types.MsgProviderUpdateAccessPolicy{
		Creator: f.member.Creator, ProviderId: providerID, AccessPolicy: types.ProviderAccessPolicy_guildMarket,
	})
	require.NoError(t, err, "the installed child-object rank grant must authorize the member")
	_, err = f.ms.GuildMembershipKick(f.ctx, &types.MsgGuildMembershipKick{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, PlayerId: f.member.Id,
	})
	require.NoError(t, err)
	// The register remains installed. Denial must come from live membership,
	// not deleting every child's grant while processing the kick.
	register := f.k.ReadGuildRankRegister(f.ctx, providerID, f.guild.Id)
	require.Contains(t, register[:], f.member.GuildRank)
	before, found := f.k.GetProvider(f.ctx, providerID)
	require.True(t, found)
	eventsBefore := len(f.ctx.EventManager().Events())
	_, err = f.ms.ProviderUpdateAccessPolicy(f.ctx, &types.MsgProviderUpdateAccessPolicy{
		Creator: f.member.Creator, ProviderId: providerID, AccessPolicy: types.ProviderAccessPolicy_openMarket,
	})
	require.Error(t, err)
	after, found := f.k.GetProvider(f.ctx, providerID)
	require.True(t, found)
	require.Equal(t, before, after)
	for _, event := range f.ctx.EventManager().Events()[eventsBefore:] {
		require.NotEqual(t, "structs.structs.EventProvider", event.Type)
	}
	_, err = f.ms.ProviderUpdateAccessPolicy(f.ctx, &types.MsgProviderUpdateAccessPolicy{
		Creator: f.buyer.Creator, ProviderId: providerID, AccessPolicy: types.ProviderAccessPolicy_openMarket,
	})
	require.NoError(t, err, "the non-member guild owner retains control after dismissing a member")
}

// Characterize the existing delegation policy, not a claim that dismissal
// revokes every kind of child-object authority. A rank holder can materialize
// their permission as a direct grant under the current generic grant handler.
func TestGuildProviderAdministratorDirectGrantSurvivesKickUntilRevoked(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	allocation, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator, SourceObjectId: f.guild.Id, Controller: f.guild.Id, AllocationType: types.AllocationType_dynamic, Power: 100})
	require.NoError(t, err)
	station, err := f.ms.SubstationCreate(f.ctx, &types.MsgSubstationCreate{Creator: f.buyer.Creator, Owner: f.guild.Id, AllocationId: allocation.AllocationId})
	require.NoError(t, err)
	_, err = f.ms.ProviderCreate(f.ctx, &types.MsgProviderCreate{
		Creator: f.buyer.Creator, OwnerId: f.guild.Id, SubstationId: station.SubstationId,
		Rate: sdk.NewInt64Coin("ualpha", 1), AccessPolicy: types.ProviderAccessPolicy_openMarket,
		CapacityMinimum: 1, CapacityMaximum: 100, DurationMinimum: 1, DurationMaximum: 100,
		ProviderCancellationPenalty: math.LegacyZeroDec(), ConsumerCancellationPenalty: math.LegacyZeroDec(),
	})
	require.NoError(t, err)
	providers := f.k.GetAllProvider(f.ctx)
	require.Len(t, providers, 1)
	providerID := providers[0].Id
	_, err = f.ms.PermissionGuildRankSet(f.ctx, &types.MsgPermissionGuildRankSet{
		Creator: f.buyer.Creator, ObjectId: providerID, GuildId: f.guild.Id,
		Permission: uint64(types.PermUpdate), Rank: f.member.GuildRank,
	})
	require.NoError(t, err)
	_, err = f.ms.PermissionGrantOnObject(f.ctx, &types.MsgPermissionGrantOnObject{
		Creator: f.buyer.Creator, ObjectId: providerID, PlayerId: f.member.Id,
		Permissions: uint64(types.PermUpdate),
	})
	require.NoError(t, err, "the rank holder can grant its own bit directly")
	_, err = f.ms.GuildMembershipKick(f.ctx, &types.MsgGuildMembershipKick{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, PlayerId: f.member.Id,
	})
	require.NoError(t, err)
	member := f.k.NewCurrentContext(f.ctx).GetPlayer(f.member.Id)
	require.NotEqual(t, f.guild.Id, member.GetGuildId())
	permissionID := keeperlib.GetObjectPermissionIDBytes(providerID, f.member.Id)
	require.True(t, f.k.NewCurrentContext(f.ctx).PermissionHasAll(permissionID, types.PermUpdate))
	_, err = f.ms.ProviderUpdateAccessPolicy(f.ctx, &types.MsgProviderUpdateAccessPolicy{
		Creator: f.member.Creator, ProviderId: providerID, AccessPolicy: types.ProviderAccessPolicy_guildMarket,
	})
	require.NoError(t, err, "direct authority remains usable after dismissal")
	_, err = f.ms.GuildUpdateOwnerId(f.ctx, &types.MsgGuildUpdateOwnerId{
		Creator: f.buyer.Creator, GuildId: f.guild.Id, Owner: f.seller.Id,
	})
	require.NoError(t, err)
	_, err = f.ms.PermissionRevokeOnObject(f.ctx, &types.MsgPermissionRevokeOnObject{
		Creator: f.seller.Creator, ObjectId: providerID, PlayerId: f.member.Id,
		Permissions: uint64(types.PermUpdate),
	})
	require.NoError(t, err, "the new guild owner can revoke the child-object grant")
	require.False(t, f.k.NewCurrentContext(f.ctx).PermissionHasAll(permissionID, types.PermUpdate))
	before, found := f.k.GetProvider(f.ctx, providerID)
	require.True(t, found)
	_, err = f.ms.ProviderUpdateAccessPolicy(f.ctx, &types.MsgProviderUpdateAccessPolicy{
		Creator: f.member.Creator, ProviderId: providerID, AccessPolicy: types.ProviderAccessPolicy_openMarket,
	})
	require.Error(t, err)
	after, found := f.k.GetProvider(f.ctx, providerID)
	require.True(t, found)
	require.Equal(t, before, after, "the refused update cannot mutate provider policy")
}
