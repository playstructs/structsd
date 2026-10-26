package app_test

import (
	"encoding/binary"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/app"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func setupGeneratorQueueApp(t *testing.T, count int) (*app.App, sdk.Context, string, string, []types.Player) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	return setupGeneratorQueueOnApp(t, bApp, ctx, count)
}

func setupGeneratorQueueOnApp(t *testing.T, bApp *app.App, ctx sdk.Context, count int) (*app.App, sdk.Context, string, string, []types.Player) {
	t.Helper()
	primary, _, ownerID := newTransferPlayer(t, bApp, ctx)


	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(3000000))
	const generatorID, planetID = "5-99501", "2-99501"
	bApp.StructsKeeper.SetPlanet(ctx, types.Planet{Id: planetID, Owner: ownerID, Creator: primary.String()})
	bApp.StructsKeeper.SetStruct(ctx, types.Struct{Id: generatorID, Owner: ownerID, Creator: primary.String(), Type: 20, LocationId: planetID, LocationType: types.ObjectType_planet})
	bApp.StructsKeeper.SetStructCount(ctx, 99502)
	bApp.StructsKeeper.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, generatorID), uint64(types.StructStateMaterialized|types.StructStateBuilt|types.StructStateOnline))
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	cc.GetPlayer(ownerID).StructsLoadIncrement(cc.GetStruct(generatorID).GetStructType().PassiveDraw)
	cc.SetStructAttribute(keeperlib.GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, ownerID, 20), 1)
	cc.CommitAll()
	players := addGeneratorQueueDonors(t, bApp, ctx, generatorID, 1, count)
	return bApp, ctx, generatorID, primary.String(), players
}

func TestGeneratorInfuseRejectsOverflowBeforeBankMovement(t *testing.T) {
	for _, scenario := range []struct {
		name, amount, reason string
	}{
		{"amount", "18446744073709551616", "fuel_overflow"},
		{"total-fuel", "18446744073709551615", "fuel_overflow"},
		{"total-power", "9223372036854774808", "power_overflow"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			bApp, ctx, generatorID, _, donors := setupGeneratorQueueApp(t, 1)
			vault, err := types.VaultAddress(donors[0].Id)
			require.NoError(t, err)
			amount, ok := math.NewIntFromString(scenario.amount)
			require.True(t, ok)
			fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewCoin("ualpha", amount)))
			before := bApp.BankKeeper.GetAllBalances(ctx, vault)
			supply := bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount
			infusions := bApp.StructsKeeper.GetAllInfusion(ctx)
			_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).StructGeneratorInfuse(ctx, &types.MsgStructGeneratorInfuse{Creator: donors[0].Creator, StructId: generatorID, InfuseAmount: scenario.amount + "ualpha"})
			require.ErrorContains(t, err, scenario.reason)
			require.Equal(t, before, bApp.BankKeeper.GetAllBalances(ctx, vault), "direct handler invocation has no transaction rollback to hide a debit")
			require.Equal(t, supply, bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount)
			require.Equal(t, infusions, bApp.StructsKeeper.GetAllInfusion(ctx))
		})
	}
}

func TestGeneratorInfuseRestoresDisabledPositionAfterActivation(t *testing.T) {
	bApp, ctx, generatorID, _, donors := setupGeneratorQueueApp(t, 1)
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	cc.GetStruct(generatorID).GoOffline()
	cc.CommitAll()
	bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
	require.Zero(t, bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(donors[0].Id).GetCapacity())
	cc = bApp.StructsKeeper.NewCurrentContext(ctx)
	cc.GetStruct(generatorID).GoOnline()
	cc.CommitAll()
	vault, err := types.VaultAddress(donors[0].Id)
	require.NoError(t, err)
	fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1)))
	_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).StructGeneratorInfuse(ctx, &types.MsgStructGeneratorInfuse{Creator: donors[0].Creator, StructId: generatorID, InfuseAmount: "1ualpha"})
	require.NoError(t, err)
	row, found := bApp.StructsKeeper.GetInfusion(ctx, generatorID, vault.String())
	require.True(t, found)
	require.False(t, row.GeneratorDisabled)
	require.Equal(t, uint64(1001), row.Fuel)
	require.Equal(t, uint64(2002), row.Power)
	bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
	require.Equal(t, uint64(2002), bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(donors[0].Id).GetCapacity(), "queued restoration must not credit the top-up twice")
}

func TestGeneratorPowerQueueCleansLegacyAllocationSources(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing=%t", missing), func(t *testing.T) {
			bApp, ctx, generatorID, _, _ := setupGeneratorQueueApp(t, 0)
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			ownerID := cc.GetStruct(generatorID).GetOwnerId()
			bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: "4-99501", Owner: ownerID})
			cc.GenesisImportAllocation(types.Allocation{Id: "6-99501", SourceObjectId: generatorID, DestinationId: "4-99501", Controller: ownerID, Type: types.AllocationType_dynamic}, 100)
			// This live allocation belongs to another source. Its stale entry
			// under the generator must never authorize destroying that source.
			cc.GenesisImportAllocation(types.Allocation{Id: "6-00000", SourceObjectId: ownerID, DestinationId: "4-99501", Controller: ownerID, Type: types.AllocationType_dynamic}, 50)
			cc.CommitAll()
			for i := 0; i < types.GeneratorPowerReconcileBudget; i++ {
				require.NoError(t, bApp.StructsKeeper.SetAllocationSourceIndex(ctx, generatorID, fmt.Sprintf("6-%05d", i)))
			}
			if missing {
				bApp.StructsKeeper.RemoveStruct(ctx, generatorID)
			}
			bApp.StructsKeeper.RebuildGeneratorPowerQueue(ctx)
			bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
			_, queued, err := bApp.StructsKeeper.GetGeneratorPowerSnapshot(ctx, generatorID)
			require.NoError(t, err)
			require.True(t, queued, "stale index rows consume the first block's entire allowance")
			_, found := bApp.StructsKeeper.GetAllocation(ctx, "6-99501")
			require.True(t, found)
			bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
			_, queued, err = bApp.StructsKeeper.GetGeneratorPowerSnapshot(ctx, generatorID)
			require.NoError(t, err)
			require.False(t, queued)
			_, found = bApp.StructsKeeper.GetAllocation(ctx, "6-99501")
			require.False(t, found)
			_, found = bApp.StructsKeeper.GetAllocation(ctx, "6-00000")
			require.True(t, found)
			require.False(t, bApp.StructsKeeper.SourceHasAllocations(ctx, generatorID))
			require.Equal(t, uint64(50), bApp.StructsKeeper.NewCurrentContext(ctx).GetSubstation("4-99501").GetCapacity())
		})
	}
}

func addGeneratorQueueDonors(t *testing.T, bApp *app.App, ctx sdk.Context, generatorID string, first, count int) []types.Player {
	t.Helper()
	server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	var players []types.Player
	for i := first; i < first+count; i++ {
		address := make([]byte, 20)
		address[0] = 0x62
		binary.BigEndian.PutUint64(address[12:], uint64(i))
		player := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(address))
		vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(player.Id)
		require.NoError(t, err)
		fundRedelegationAccount(t, bApp, ctx, vaults[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", 1000)))
		_, err = server.StructGeneratorInfuse(ctx, &types.MsgStructGeneratorInfuse{Creator: player.Creator, StructId: generatorID, InfuseAmount: "1000ualpha"})
		require.NoError(t, err)
		players = append(players, player)
	}
	return players
}

func TestGeneratorPowerQueueBoundsShutdownRecoveryAndDestruction(t *testing.T) {
	count := types.GeneratorPowerReconcileBudget + 2
	bApp, ctx, generatorID, operator, donors := setupGeneratorQueueApp(t, count)
	ctx = ctx.WithBlockHeight(1000)
	server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	before := bApp.StructsKeeper.GetAllInfusion(ctx)
	_, err := server.StructDeactivate(ctx, &types.MsgStructDeactivate{Creator: operator, StructId: generatorID})
	require.NoError(t, err)
	require.Equal(t, before, bApp.StructsKeeper.GetAllInfusion(ctx), "the transaction may not walk donor-sized state")
	snapshot, queued, err := bApp.StructsKeeper.GetGeneratorPowerSnapshot(ctx, generatorID)
	require.NoError(t, err)
	require.True(t, queued)
	require.Equal(t, uint64(count), snapshot.Through)
	bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
	disabled := 0
	for _, donor := range donors {
		vault, err := types.VaultAddress(donor.Id)
		require.NoError(t, err)
		row, found := bApp.StructsKeeper.GetInfusion(ctx, generatorID, vault.String())
		require.True(t, found)
		require.Equal(t, uint64(1000), row.Fuel)
		require.Equal(t, uint64(2), row.Ratio)
		if row.GeneratorDisabled {
			disabled++
			require.Zero(t, row.Power)
		}
	}
	require.Equal(t, types.GeneratorPowerReconcileBudget, disabled)
	_, err = server.StructActivate(ctx, &types.MsgStructActivate{Creator: operator, StructId: generatorID})
	require.NoError(t, err, "the operator retains independent reactor power for activation")
	snapshot, queued, err = bApp.StructsKeeper.GetGeneratorPowerSnapshot(ctx, generatorID)
	require.NoError(t, err)
	require.True(t, queued)
	require.Zero(t, snapshot.After)
	require.Equal(t, uint64(count), snapshot.Through)
	for pass := 0; pass < 2; pass++ {
		late := addGeneratorQueueDonors(t, bApp, ctx, generatorID, count+pass*128+1, 128)
		donors = append(donors, late...)
		current, queued, err := bApp.StructsKeeper.GetGeneratorPowerSnapshot(ctx, generatorID)
		require.NoError(t, err)
		require.True(t, queued)
		require.Equal(t, snapshot.Through, current.Through, "arrivals cannot extend restoration")
		bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
	}
	_, queued, err = bApp.StructsKeeper.GetGeneratorPowerSnapshot(ctx, generatorID)
	require.NoError(t, err)
	require.False(t, queued)
	for _, donor := range donors {
		vault, err := types.VaultAddress(donor.Id)
		require.NoError(t, err)
		row, found := bApp.StructsKeeper.GetInfusion(ctx, generatorID, vault.String())
		require.True(t, found)
		require.False(t, row.GeneratorDisabled)
		require.Equal(t, uint64(2000), row.Power)
		require.Equal(t, uint64(2000), bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(donor.Id).GetCapacity())
	}
	supply := bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	cc.GetStruct(generatorID).DestroyAndCommit()
	cc.CommitAll()
	require.Len(t, bApp.StructsKeeper.NewCurrentContext(ctx).GetAllInfusionByDestination(generatorID), len(donors))
	for pass := 0; pass < (len(donors)+127)/128; pass++ {
		beforeCount := len(bApp.StructsKeeper.NewCurrentContext(ctx).GetAllInfusionByDestination(generatorID))
		bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
		afterCount := len(bApp.StructsKeeper.NewCurrentContext(ctx).GetAllInfusionByDestination(generatorID))
		require.Greater(t, beforeCount, afterCount)
		require.LessOrEqual(t, beforeCount-afterCount, types.GeneratorPowerReconcileBudget)
		if pass == 0 {
			bApp.StructsKeeper.RemoveStruct(ctx, generatorID)
		} // a later sweep cannot lose the remaining job
	}
	_, queued, err = bApp.StructsKeeper.GetGeneratorPowerSnapshot(ctx, generatorID)
	require.NoError(t, err)
	require.False(t, queued)
	for _, donor := range donors {
		require.Zero(t, bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(donor.Id).GetCapacity())
		vault, err := types.VaultAddress(donor.Id)
		require.NoError(t, err)
		require.True(t, bApp.BankKeeper.GetBalance(ctx, vault, "ualpha").IsZero(), "burned generator fuel is never refunded")
	}
	require.Equal(t, supply, bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount)
}
