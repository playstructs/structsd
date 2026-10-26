package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildAllocationControllerCreationAndOwnershipTransfer(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	created, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{
		Creator: f.buyer.Creator, SourceObjectId: f.guild.Id,
		Controller: f.guild.Id, AllocationType: types.AllocationType_dynamic, Power: 1,
	})
	require.NoError(t, err)
	cc := f.k.NewCurrentContext(f.ctx)
	allocation, exists := cc.GetAllocation(created.AllocationId)
	require.True(t, exists)
	require.Equal(t, f.guild.Id, allocation.GetOwnerId())
	require.Nil(t, allocation.GetOwner(), "guild control does not synthesize a personal owner")
	var indexedController string
	for _, event := range f.ctx.EventManager().ABCIEvents() {
		if event.Type != "structs.structs.EventAllocation" {
			continue
		}
		decoded, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		if row := decoded.(*types.EventAllocation).Allocation; row.Id == created.AllocationId {
			indexedController = row.Controller
		}
	}
	require.Equal(t, f.guild.Id, indexedController, "existing allocation event exposes guild control to indexers")
	for _, id := range []string{f.guild.Id, f.buyer.Id} {
		require.Equal(t, types.Permissionless, cc.GetPermissions(keeperlib.GetObjectPermissionIDBytes(created.AllocationId, id)), "no guild-as-player or creating-owner permission row")
	}
	_, err = f.ms.GuildUpdateOwnerId(f.ctx, &types.MsgGuildUpdateOwnerId{Creator: f.buyer.Creator, GuildId: f.guild.Id, Owner: f.seller.Id})
	require.NoError(t, err)
	transfer := func(address, controller string) error {
		_, err := f.ms.AllocationTransfer(f.ctx, &types.MsgAllocationTransfer{Creator: address, AllocationId: created.AllocationId, Controller: controller})
		return err
	}
	require.Error(t, transfer(f.buyer.Creator, f.buyer.Id), "former guild owner retains no implicit asset grant")
	require.Error(t, transfer(f.seller.Creator, "0-999999"))
	require.Error(t, transfer(f.seller.Creator, "0-01"))
	require.Error(t, transfer(f.seller.Creator, "6-1"))
	require.NoError(t, transfer(f.seller.Creator, f.buyer.Id))
	require.NoError(t, transfer(f.buyer.Creator, f.guild.Id), "player controller can transfer to a guild")
	require.Equal(t, types.Permissionless, f.k.NewCurrentContext(f.ctx).GetPermissions(keeperlib.GetObjectPermissionIDBytes(created.AllocationId, f.guild.Id)))
}

func TestGuildAllocationControllerRankCannotBypassSigningKey(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	created, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator, SourceObjectId: f.guild.Id, Controller: f.guild.Id, AllocationType: types.AllocationType_static, Power: 1})
	require.NoError(t, err)
	msg := &types.MsgAllocationTransfer{Creator: f.member.Creator, AllocationId: created.AllocationId, Controller: f.member.Id}
	_, err = f.ms.AllocationTransfer(f.ctx, msg)
	require.Error(t, err)
	f.k.SetGuildRankPermissionStoreOnly(f.ctx, created.AllocationId, f.guild.Id, types.PermAdmin, f.member.GuildRank)
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermAdmin)
	_, err = f.ms.AllocationTransfer(f.ctx, msg)
	require.Error(t, err)
    testPermissionAdd(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermAdmin)
    _, err = f.ms.AllocationTransfer(f.ctx, msg)
    require.Error(t, err, "rank cannot confer administrative transfer authority")
    testPermissionAdd(f.k, f.ctx, keeperlib.GetObjectPermissionIDBytes(created.AllocationId, f.member.Id), types.PermAdmin)
    _, err = f.ms.AllocationTransfer(f.ctx, msg)
    require.NoError(t, err)
}
