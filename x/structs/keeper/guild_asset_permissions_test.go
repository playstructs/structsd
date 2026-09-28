package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildAssetPermissionsFollowLiveOwnerAndKeyCeiling(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	// Seed an owned object directly: allocation controller transactions are
	// migrated separately. No individual permission rows are assigned here.
	allocation, err := f.k.SetAllocationOnly(f.ctx, types.Allocation{Id: "6-999", Controller: f.guild.Id})
	require.NoError(t, err)
	check := func(address string) error {
		cc := f.k.NewCurrentContext(f.ctx)
		actor, err := cc.GetSigningPlayer(address)
		require.NoError(t, err)
		object, exists := cc.GetAllocation(allocation.Id)
		require.True(t, exists)
		return cc.PermissionCheck(object, actor, types.PermDelete)
	}
	require.NoError(t, check(f.buyer.Creator), "non-member owner controls guild assets")
	require.Error(t, check(f.seller.Creator), "former owner has no inherited asset grant")
	require.Error(t, check(f.member.Creator), "membership alone grants no authority")
	f.k.SetGuildRankPermissionStoreOnly(f.ctx, allocation.Id, f.guild.Id, types.PermDelete, f.member.GuildRank)
	require.NoError(t, check(f.member.Creator))
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermDelete)
	require.Error(t, check(f.member.Creator))
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.buyer.Creator), types.PermDelete)
	require.Error(t, check(f.buyer.Creator), "ownership cannot bypass the key ceiling")
	testPermissionAdd(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.buyer.Creator), types.PermDelete)

	cc := f.k.NewCurrentContext(f.ctx)
	actor, err := cc.GetSigningPlayer(f.buyer.Creator)
	require.NoError(t, err)
	object, exists := cc.GetAllocation(allocation.Id)
	require.True(t, exists)
	require.NoError(t, cc.PermissionCheck(object, actor, types.PermDelete))
	cc.GetGuild(f.guild.Id).SetOwner(f.seller.Id)
	require.Error(t, cc.PermissionCheck(object, actor, types.PermDelete), "cached guild ownership changes apply before commit")
	cc.CommitAll()
	require.NoError(t, check(f.seller.Creator))
	require.Error(t, check(f.buyer.Creator))

	allocation.Controller = "0-999999"
	_, err = f.k.SetAllocationOnly(f.ctx, allocation)
	require.NoError(t, err)
	require.Error(t, check(f.buyer.Creator), "a missing guild does not establish ownership")
}
