package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestReactorPowerMigrationIsAtomicIdempotentAndDisablesInactivePower(t *testing.T) {
	for _, jailed := range []bool{false, true} {
		name := "active"
		if jailed {
			name = "jailed"
		}
		t.Run(name, func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 2)
			primary, _, playerID := newTransferPlayer(t, bApp, ctx)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			for _, validator := range validators {
				address, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
				require.NoError(t, err)
				delegate(t, bApp, ctx, server, denom, primary, address, math.NewInt(1000000))
			}
			keepers := delegationUpgradeKeepers(bApp)
			require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
			stakeAddress := primary
			if jailed {
				address, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
				require.NoError(t, err)
				validator, err := bApp.StakingKeeper.GetValidator(ctx, address)
				require.NoError(t, err)
				validator.Jailed = true
				require.NoError(t, bApp.StakingKeeper.SetValidator(ctx, validator))
			}
			// Sort the orphan after real reactors, so failure occurs after earlier
			// positions have committed into the migration's private cache.
			orphan := types.Infusion{DestinationId: "zz-missing-reactor", DestinationType: types.ObjectType_reactor, Address: primary.String(), PlayerId: playerID}
			bApp.StructsKeeper.SetInfusion(ctx, orphan)
			unchanged := bApp.StructsKeeper.GetAllInfusion(ctx)
			unchangedCapacity := redelegationCapacity(t, bApp, ctx, playerID)
			unchangedEvents := len(ctx.EventManager().Events())
			require.ErrorContains(t, v22.MigrateReactorPower(ctx, keepers), "missing reactor")
			require.Equal(t, unchanged, bApp.StructsKeeper.GetAllInfusion(ctx))
			require.Equal(t, unchangedCapacity, redelegationCapacity(t, bApp, ctx, playerID))
			require.Len(t, ctx.EventManager().Events(), unchangedEvents)
			bApp.StructsKeeper.RemoveInfusion(ctx, orphan.DestinationId, orphan.Address)
			require.NoError(t, v22.MigrateReactorPower(ctx, keepers))
			for i, validator := range validators {
				address, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
				require.NoError(t, err)
				active := !(jailed && i == 0)
				row, found := bApp.StructsKeeper.GetInfusion(ctx, jailGateReactorId(t, bApp, ctx, address), stakeAddress.String())
				require.True(t, found)

				power := uint64(1000000)
				if !active {
					power = 0
				}
				require.Equal(t, power, row.Power)
			}
			before := bApp.StructsKeeper.GetAllInfusion(ctx)
			capacity := redelegationCapacity(t, bApp, ctx, playerID)
			eventCount := len(ctx.EventManager().Events())
			require.NoError(t, v22.MigrateReactorPower(ctx, keepers))
			require.Equal(t, before, bApp.StructsKeeper.GetAllInfusion(ctx), "rerun preserves backing and power")
			require.Equal(t, capacity, redelegationCapacity(t, bApp, ctx, playerID))
			require.Len(t, ctx.EventManager().Events(), eventCount)
		})
	}
}
