package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func seedGuildInfusion(t *testing.T, f *nonMemberOwnerFixture) string {
	t.Helper()
	cc := f.k.NewCurrentContext(f.ctx)
	vaults, err := cc.EnsureVaults(f.guild.Id)
	require.NoError(t, err)
	reactor, found := f.k.GetReactor(f.ctx, f.guild.PrimaryReactorId)
	require.True(t, found)
	validator, err := sdk.ValAddressFromBech32(reactor.Validator)
	require.NoError(t, err)
	mock := f.k.StakingKeeper().(*keepertest.MockStakingKeeper)
	mock.AddValidator(validator, math.NewInt(1000))
	require.NoError(t, mock.SetDelegation(f.ctx, stakingtypes.NewDelegation(vaults[0].String(), validator.String(), math.LegacyNewDec(1000))))
	infusion, err := cc.UpsertOwnedInfusion(types.ObjectType_reactor, f.guild.PrimaryReactorId, vaults[0].String(), f.guild.Id)
	require.NoError(t, err)
	infusion.SetRatio(1)
	infusion.SetFuelAndCommission(1000, math.LegacyZeroDec())
	cc.CommitAll()
	return vaults[0].String()
}

func TestGuildAllocationSourceUsesOwnerRankAndSigningKeyPermissions(t *testing.T) {
	f := newNonMemberOwnerFixture(t)
	seedGuildInfusion(t, f)
	create := func(creator, source string) error {
		_, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: creator, SourceObjectId: source, AllocationType: types.AllocationType_static, Power: 1})
		return err
	}
	require.NoError(t, create(f.buyer.Creator, f.guild.Id), "owner need not belong to the guild")
	require.Error(t, create(f.member.Creator, f.guild.Id), "membership alone grants no access to guild power")
	f.k.SetGuildRankPermissionStoreOnly(f.ctx, f.guild.Id, f.guild.Id, types.PermSourceAllocation, f.member.GuildRank)
	require.NoError(t, create(f.member.Creator, f.guild.Id))
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.member.Creator), types.PermSourceAllocation)
	require.Error(t, create(f.member.Creator, f.guild.Id), "rank cannot exceed signing-key permissions")
	testPermissionRemove(f.k, f.ctx, keeperlib.GetAddressPermissionIDBytes(f.buyer.Creator), types.PermSourceAllocation)
	require.Error(t, create(f.buyer.Creator, f.guild.Id), "ownership cannot exceed signing-key permissions")
	require.Error(t, create(f.seller.Creator, "0-999999"), "a cache handle is not an existing guild")
}

func TestGuildInfusionCapacityCutResizesOrCascadesAllocations(t *testing.T) {
	for _, allocationType := range []types.AllocationType{types.AllocationType_static, types.AllocationType_automated} {
		t.Run(allocationType.String(), func(t *testing.T) {
			f := newNonMemberOwnerFixture(t)
			address := seedGuildInfusion(t, f)
			created, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator, SourceObjectId: f.guild.Id, AllocationType: allocationType, Power: 800})
			require.NoError(t, err)
			station, err := f.ms.SubstationCreate(f.ctx, &types.MsgSubstationCreate{Creator: f.buyer.Creator, AllocationId: created.AllocationId})
			require.NoError(t, err)
			cc := f.k.NewCurrentContext(f.ctx)
			cc.GetInfusion(f.guild.PrimaryReactorId, address).SetFuel(400)
			cc.CommitAll()
			_, err = f.k.EndBlocker(f.ctx)
			require.NoError(t, err)
			load := f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, f.guild.Id))
			capacity := f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, station.SubstationId))
			allocation, exists := f.k.NewCurrentContext(f.ctx).GetAllocation(created.AllocationId)
			if allocationType == types.AllocationType_automated {
				require.True(t, exists)
				require.Equal(t, uint64(400), allocation.GetPower())
				require.Equal(t, uint64(400), load)
				require.Equal(t, uint64(400), capacity)
			} else {
				require.False(t, exists)
				require.Zero(t, load)
				require.Zero(t, capacity)
				require.Empty(t, f.k.GetAllAllocationIdBySourceIndex(f.ctx, f.guild.Id))
			}
		})
	}
}

func TestGuildAllocationRankCannotPrivatizePowerAndOwnerCanReclaim(t *testing.T) {
	for _, kind := range []types.AllocationType{types.AllocationType_static, types.AllocationType_automated} {
		t.Run(kind.String(), func(t *testing.T) {
			f := newNonMemberOwnerFixture(t)
			seedGuildInfusion(t, f)
			f.k.SetGuildRankPermissionStoreOnly(f.ctx, f.guild.Id, f.guild.Id, types.PermSourceAllocation, f.member.GuildRank)
			before := f.k.GetAllocationCount(f.ctx)
			_, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.member.Creator, SourceObjectId: f.guild.Id, Controller: f.member.Id, AllocationType: kind, Power: 1})
			require.Error(t, err)
			require.Equal(t, before, f.k.GetAllocationCount(f.ctx))
			created, err := f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.member.Creator, SourceObjectId: f.guild.Id, AllocationType: kind, Power: 1})
			require.NoError(t, err)
			allocation, found := f.k.GetAllocation(f.ctx, created.AllocationId)
			require.True(t, found)
			require.Equal(t, f.guild.Id, allocation.Controller)
			require.Zero(t, f.k.GetPermissionsByBytes(f.ctx, keeperlib.GetObjectPermissionIDBytes(created.AllocationId, f.member.Id)))
			_, err = f.ms.AllocationDelete(f.ctx, &types.MsgAllocationDelete{Creator: f.member.Creator, AllocationId: created.AllocationId})
			require.Error(t, err)
			_, err = f.ms.GuildMembershipKick(f.ctx, &types.MsgGuildMembershipKick{Creator: f.buyer.Creator, GuildId: f.guild.Id, PlayerId: f.member.Id})
			require.NoError(t, err)
			_, err = f.ms.AllocationDelete(f.ctx, &types.MsgAllocationDelete{Creator: f.buyer.Creator, AllocationId: created.AllocationId})
			require.NoError(t, err)
			_, found = f.k.GetAllocation(f.ctx, created.AllocationId)
			require.False(t, found)
			// Explicit administration can assign a personal controller; even then the
			// guild owner retains the ability to reclaim the source's committed power.
			created, err = f.ms.AllocationCreate(f.ctx, &types.MsgAllocationCreate{Creator: f.buyer.Creator, SourceObjectId: f.guild.Id, Controller: f.member.Id, AllocationType: kind, Power: 1})
			require.NoError(t, err)
			_, err = f.ms.AllocationDelete(f.ctx, &types.MsgAllocationDelete{Creator: f.buyer.Creator, AllocationId: created.AllocationId})
			require.NoError(t, err)
		})
	}
}
