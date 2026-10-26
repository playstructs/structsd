package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildSubstationOwnershipAndCosmeticPermissions(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	created, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator, SourceObjectId: f.guild.Id, Controller: f.guild.Id, AllocationType: types.AllocationType_dynamic, Power: 100})
	require.NoError(t, err)
	for _, owner := range []string{"0-999999", "0-01", "6-1"} {
		_, err = f.ms.SubstationCreate(f.ctx, &types.MsgSubstationCreate{Creator: f.buyer.Creator, Owner: owner, AllocationId: created.AllocationId})
		require.Error(t, err)
	}
	station, err := f.ms.SubstationCreate(f.ctx, &types.MsgSubstationCreate{Creator: f.buyer.Creator, Owner: f.guild.Id, AllocationId: created.AllocationId})
	require.NoError(t, err)
	cc := f.k.NewCurrentContext(f.ctx)
	cache := cc.GetSubstation(station.SubstationId)
	require.Equal(t, f.guild.Id, cache.GetOwnerId())
	require.Nil(t, cache.GetOwner())
	for _, id := range []string{f.guild.Id, f.buyer.Id} {
		require.Equal(t, types.Permissionless, cc.GetPermissions(keeperlib.GetObjectPermissionIDBytes(station.SubstationId, id)))
	}
	update := func(address string) error {
		_, err := f.ms.SubstationUpdateName(f.ctx, &types.MsgSubstationUpdateName{Creator: address, SubstationId: station.SubstationId, Name: "Guild grid"})
		return err
	}
	require.NoError(t, update(f.buyer.Creator))
	require.Error(t, update(f.member.Creator))
	f.k.SetGuildRankPermissionStoreOnly(f.ctx, f.guild.Id, f.guild.Id, types.PermGuildUGCUpdate, f.member.GuildRank)
	require.NoError(t, update(f.member.Creator))
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermGuildUGCUpdate)
	require.Error(t, update(f.member.Creator), "cosmetic rank cannot exceed the signing key")
	_, err = f.ms.GuildUpdateOwnerId(f.ctx, &types.MsgGuildUpdateOwnerId{Creator: f.buyer.Creator, GuildId: f.guild.Id, Owner: f.seller.Id})
	require.NoError(t, err)
	require.Error(t, update(f.buyer.Creator), "creating owner retains no automatic personal permission")
	require.NoError(t, update(f.seller.Creator))
	_, err = f.ms.SubstationDelete(f.ctx, &types.MsgSubstationDelete{Creator: f.buyer.Creator, SubstationId: station.SubstationId})
	require.Error(t, err)
	_, err = f.ms.SubstationDelete(f.ctx, &types.MsgSubstationDelete{Creator: f.seller.Creator, SubstationId: station.SubstationId})
	require.NoError(t, err)
	require.Error(t, f.k.NewCurrentContext(f.ctx).GetSubstation(station.SubstationId).CheckSubstation())
}

func TestGuildSubstationCreationRequiresGuildAuthority(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	created, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator, SourceObjectId: f.guild.Id, Controller: f.member.Id, AllocationType: types.AllocationType_static, Power: 1})
	require.NoError(t, err)
	msg := &types.MsgSubstationCreate{Creator: f.member.Creator, Owner: f.guild.Id, AllocationId: created.AllocationId}
	_, err = f.ms.SubstationCreate(f.ctx, msg)
	require.Error(t, err, "allocation control does not authorize creating assets for another guild")
	f.k.SetGuildRankPermissionStoreOnly(f.ctx, f.guild.Id, f.guild.Id, types.PermAllocationConnection, f.member.GuildRank)
	_, err = f.ms.SubstationCreate(f.ctx, msg)
	require.NoError(t, err)
}

func TestGuildEntrySubstationRankAuthorityFollowsMembershipAndKey(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	allocation, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator,
		SourceObjectId: f.guild.Id, Controller: f.guild.Id, AllocationType: types.AllocationType_dynamic, Power: 100})
	require.NoError(t, err)
	station, err := f.ms.SubstationCreate(f.ctx, &types.MsgSubstationCreate{Creator: f.buyer.Creator,
		Owner: f.guild.Id, AllocationId: allocation.AllocationId})
	require.NoError(t, err)
	grant := func(object string, bit types.Permission) {
		t.Helper()
		_, err := f.ms.PermissionGuildRankSet(f.ctx, &types.MsgPermissionGuildRankSet{Creator: f.buyer.Creator,
			ObjectId: object, GuildId: f.guild.Id, Permission: uint64(bit), Rank: 5})
		require.NoError(t, err)
	}
	attempt := func(allowed bool) {
		t.Helper()
		before, found := f.k.GetGuild(f.ctx, f.guild.Id)
		require.True(t, found)
		f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
		_, err := f.ms.GuildUpdateEntrySubstationId(f.ctx, &types.MsgGuildUpdateEntrySubstationId{
			Creator: f.member.Creator, GuildId: f.guild.Id, EntrySubstationId: station.SubstationId})
		after, found := f.k.GetGuild(f.ctx, f.guild.Id)
		require.True(t, found)
		if allowed {
			require.NoError(t, err)
			require.Equal(t, station.SubstationId, after.EntrySubstationId)
		} else {
			require.ErrorContains(t, err, "permission")
			require.Equal(t, before, after)
			for _, event := range f.ctx.EventManager().ABCIEvents() {
				require.NotEqual(t, "structs.structs.EventGuild", event.Type)
			}
		}
		for _, object := range []string{f.guild.Id, station.SubstationId} {
			require.Equal(t, types.Permissionless, f.k.GetPermissionsByBytes(f.ctx,
				keeperlib.GetObjectPermissionIDBytes(object, f.member.Id)), "using a rank grant does not create a persistent personal grant")
		}
	}
	attempt(false)
	grant(f.guild.Id, types.PermGuildSubstationUpdate)
	attempt(false)
	grant(station.SubstationId, types.PermSubstationConnection)
	attempt(true)
	keyID := keeperlib.GetAddressPermissionIDBytes(f.member.Creator)
	testPermissionRemove(f.k, f.ctx, keyID, types.PermSubstationConnection)
	attempt(false)
	testPermissionAdd(f.k, f.ctx, keyID, types.PermSubstationConnection)
	_, err = f.ms.PlayerUpdateGuildRank(f.ctx, &types.MsgPlayerUpdateGuildRank{Creator: f.buyer.Creator,
		GuildId: f.guild.Id, PlayerId: f.member.Id, GuildRank: 6})
	require.NoError(t, err)
	attempt(false)
	_, err = f.ms.PlayerUpdateGuildRank(f.ctx, &types.MsgPlayerUpdateGuildRank{Creator: f.buyer.Creator,
		GuildId: f.guild.Id, PlayerId: f.member.Id, GuildRank: 5})
	require.NoError(t, err)
	attempt(true)
	_, err = f.ms.PermissionGuildRankRevoke(f.ctx, &types.MsgPermissionGuildRankRevoke{Creator: f.buyer.Creator,
		ObjectId: station.SubstationId, GuildId: f.guild.Id, Permission: uint64(types.PermSubstationConnection)})
	require.NoError(t, err)
	attempt(false)
	grant(station.SubstationId, types.PermSubstationConnection)
	_, err = f.ms.GuildMembershipKick(f.ctx, &types.MsgGuildMembershipKick{Creator: f.buyer.Creator,
		GuildId: f.guild.Id, PlayerId: f.member.Id})
	require.NoError(t, err)
	attempt(false)
}
