package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildAgreementFundingExtensionAndOwnershipTransfer(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0", "0")
	guild := types.Guild{Id: "0-900", Owner: f.consumer.Id}
	f.k.SetGuild(f.ctx, guild)
	vaults, err := f.k.NewCurrentContext(f.ctx).EnsureVaults(guild.Id)
	require.NoError(t, err)
	f.fund(t, vaults[0], 60000)
	f.fund(t, f.consumerAcc, 777)
	msg := &types.MsgAgreementOpen{Creator: f.consumer.Creator, OwnerId: guild.Id, ProviderId: f.provider.Id, Capacity: 100, Duration: 50}
	for _, owner := range []string{"0-999999", "0-01", "6-1"} {
		msg.OwnerId = owner
		_, err = f.ms.AgreementOpen(f.ctx, msg)
		require.Error(t, err)
	}
	msg.OwnerId = guild.Id
	msg.Creator = f.provider.Creator
	_, err = f.ms.AgreementOpen(f.ctx, msg)
	require.Error(t, err, "provider ownership is not authority to spend a consumer guild's money")
	msg.Creator = f.consumer.Creator
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermTokenTransfer)
	_, err = f.ms.AgreementOpen(f.ctx, msg)
	require.Error(t, err)
	testPermissionAdd(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(msg.Creator), types.PermTokenTransfer)
	_, err = f.ms.AgreementOpen(f.ctx, msg)
	require.NoError(t, err)
	ids := f.k.GetAllAgreementIdByProviderIndex(f.ctx, f.provider.Id)
	require.Len(t, ids, 1)
	agreement, found := f.k.GetAgreement(f.ctx, ids[0])
	require.True(t, found)
	require.Equal(t, guild.Id, agreement.Owner)
	cc := f.k.NewCurrentContext(f.ctx)
	allocation, found := cc.GetAllocation(agreement.AllocationId)
	require.True(t, found)
	require.Equal(t, guild.Id, allocation.GetOwnerId())
	for _, object := range []string{agreement.Id, allocation.ID()} {
		for _, owner := range []string{guild.Id, f.consumer.Id} {
			require.Equal(t, types.Permissionless, cc.GetPermissions(keeperlib.GetObjectPermissionIDBytes(object, owner)))
		}
	}
	require.Equal(t, math.NewInt(10000), f.balance(vaults[0]))
	require.Equal(t, math.NewInt(777), f.balance(f.consumerAcc))
	// An update grant plus a full signing key still does not authorize spending
	// from the guild vault. Treasury permission is a separate object check.
	testPermissionAdd(f.k, f.ctx, keeperlib.GetObjectPermissionIDBytes(agreement.Id, f.provider.Owner), types.PermUpdate)
	extend := &types.MsgAgreementDurationIncrease{Creator: f.provider.Creator, AgreementId: agreement.Id, DurationIncrease: 10}
	_, err = f.ms.AgreementDurationIncrease(f.ctx, extend)
	require.Error(t, err)
	require.Equal(t, math.NewInt(10000), f.balance(vaults[0]))
	extend.Creator = f.consumer.Creator
	_, err = f.ms.AgreementDurationIncrease(f.ctx, extend)
	require.NoError(t, err)
	require.True(t, f.balance(vaults[0]).IsZero())
	cc = f.k.NewCurrentContext(f.ctx)
	cc.GetGuild(guild.Id).SetOwner(f.provider.Owner)
	cc.CommitAll()
	_, err = f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{Creator: f.consumer.Creator, AgreementId: agreement.Id})
	require.Error(t, err)
	_, err = f.ms.AgreementClose(f.ctx, &types.MsgAgreementClose{Creator: f.provider.Creator, AgreementId: agreement.Id})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(60000), f.balance(vaults[0]))
	require.Equal(t, math.NewInt(777), f.balance(f.consumerAcc))
	f.requireSettledOnce(t, agreement)
}
