package v0_17_0_test

import (
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"structs/app/upgrades"
	v0_17_0 "structs/app/upgrades/v0_17_0"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

// TestMigrateDefusingInfusions_RecoversStuckInfusion is the regression test
// for the user-reported stuck infusion in incident 2026-05-defusing: an
// infusion has Defusing > 0 in store, but the underlying Cosmos UBD has
// already matured silently. The upgrade should reconcile Defusing back to
// zero and leave no maturity-queue rows behind.
func TestMigrateDefusingInfusions_RecoversStuckInfusion(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	playerAcc := sdk.AccAddress("upgrade1_padding___________________01")
	playerObj := types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()}
	playerObj.Index = k.GetPlayerCount(ctx)
	playerObj.Id = "2-" + strconv.FormatUint(playerObj.Index, 10)
	k.SetPlayer(ctx, playerObj)
	k.SetPlayerCount(ctx, playerObj.Index+1)
	k.SetPlayerIndexForAddress(ctx, playerObj.PrimaryAddress, playerObj.Index)

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	mock.AddValidator(validatorAddress, math.NewInt(1_000_000))

	reactor := k.AppendReactor(ctx, types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	})

	// Seed the bug-state: infusion.Defusing > 0 but no UBD on the staking side.
	k.SetInfusion(ctx, types.Infusion{
		DestinationType: types.ObjectType_reactor,
		DestinationId:   reactor.Id,
		Address:         playerAcc.String(),
		Defusing:        500,
		Commission:      math.LegacyZeroDec(),
	})

	keepers := &upgrades.Keepers{StructsKeeper: k}
	require.NoError(t, v0_17_0.MigrateDefusingInfusions(ctx, keepers))

	got, found := k.GetInfusion(ctx, reactor.Id, playerAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(0), got.Defusing, "stuck Defusing should be cleared")
	require.Empty(t, k.GetInfusionMaturitySweepQueueExport(ctx), "no UBD means no queue rows to bootstrap")
}

// TestMigrateDefusingInfusions_BootstrapsQueueForLiveUBD verifies that when a
// defusing infusion still has live UBD entries (because some are not yet
// mature), the upgrade bootstraps the new InfusionMaturitySweepQueue from
// each entry's CompletionTime so the EndBlocker keeps reconciling them
// post-upgrade.
func TestMigrateDefusingInfusions_BootstrapsQueueForLiveUBD(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	playerAcc := sdk.AccAddress("upgrade2_padding___________________02")
	playerObj := types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()}
	playerObj.Index = k.GetPlayerCount(ctx)
	playerObj.Id = "2-" + strconv.FormatUint(playerObj.Index, 10)
	k.SetPlayer(ctx, playerObj)
	k.SetPlayerCount(ctx, playerObj.Index+1)
	k.SetPlayerIndexForAddress(ctx, playerObj.PrimaryAddress, playerObj.Index)

	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	mock.AddValidator(validatorAddress, math.NewInt(1_000_000))

	reactor := k.AppendReactor(ctx, types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	})

	k.SetInfusion(ctx, types.Infusion{
		DestinationType: types.ObjectType_reactor,
		DestinationId:   reactor.Id,
		Address:         playerAcc.String(),
		Defusing:        300,
		Commission:      math.LegacyZeroDec(),
	})

	completion1 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	completion2 := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	require.NoError(t, mock.SetUnbondingDelegation(ctx, stakingtypes.UnbondingDelegation{
		DelegatorAddress: playerAcc.String(),
		ValidatorAddress: validatorAddress.String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			{CompletionTime: completion1, Balance: math.NewInt(100), InitialBalance: math.NewInt(100), UnbondingId: 1},
			{CompletionTime: completion2, Balance: math.NewInt(200), InitialBalance: math.NewInt(200), UnbondingId: 2},
		},
	}))

	keepers := &upgrades.Keepers{StructsKeeper: k}
	require.NoError(t, v0_17_0.MigrateDefusingInfusions(ctx, keepers))

	queue := k.GetInfusionMaturitySweepQueueExport(ctx)
	require.Len(t, queue, 2, "one queue row per UBD entry")

	got, _ := k.GetInfusion(ctx, reactor.Id, playerAcc.String())
	require.Equal(t, uint64(300), got.Defusing, "live UBD totals should round-trip through reconciliation")
}

// TestMigrateDefusingInfusions_SkipsHealthyInfusions confirms that infusions
// with Defusing == 0 (i.e. healthy or non-defusing) are not touched by the
// upgrade — no extra queue rows, no spurious reconciliation calls.
func TestMigrateDefusingInfusions_SkipsHealthyInfusions(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	playerAcc := sdk.AccAddress("upgrade3_padding___________________03")
	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	mock.AddValidator(validatorAddress, math.NewInt(1_000_000))

	reactor := k.AppendReactor(ctx, types.Reactor{
		Validator:  validatorAddress.String(),
		RawAddress: validatorAddress.Bytes(),
	})

	k.SetInfusion(ctx, types.Infusion{
		DestinationType: types.ObjectType_reactor,
		DestinationId:   reactor.Id,
		Address:         playerAcc.String(),
		Fuel:            500,
		Commission:      math.LegacyZeroDec(),
	})

	keepers := &upgrades.Keepers{StructsKeeper: k}
	require.NoError(t, v0_17_0.MigrateDefusingInfusions(ctx, keepers))

	require.Empty(t, k.GetInfusionMaturitySweepQueueExport(ctx), "healthy infusions should not produce queue rows")
}
