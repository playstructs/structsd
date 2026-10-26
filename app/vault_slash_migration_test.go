package app_test
import (
 "testing"
 "cosmossdk.io/math"
 sdk "github.com/cosmos/cosmos-sdk/types"
 stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
 stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
 "github.com/stretchr/testify/require"
 v22 "structs/app/upgrades/v0_22_0"
)
// Native-sized records retain SDK eligibility and exact burn behavior after upgrade.
func TestRetainedUnbondingSlashEligibility(t *testing.T) {
	for _, tc := range []struct {
		name      string
		factor    string
		mature    bool
		held      bool
		tooRecent bool
		prior     string
		nominal   int64
		burn      int64
	}{
		{name: "ordinary", factor: "0.1", nominal: 20000, burn: 20000},
		{name: "zero factor", factor: "0"},
		{name: "rounded zero", factor: "0.000001"},
		{name: "later infraction", factor: "0.1", tooRecent: true},
		{name: "mature", factor: "0.1", mature: true},
		{name: "mature held", factor: "0.1", mature: true, held: true, nominal: 20000, burn: 20000},
		{name: "held before infraction", factor: "0.1", mature: true, held: true, tooRecent: true},
		{name: "previously slashed", factor: "0.1", prior: "0.99", nominal: 20000, burn: 2000},
		{name: "empty balances", factor: "0.1", prior: "1", nominal: 20000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			primary, _, _ := newTransferPlayer(t, bApp, ctx)
			server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			for _, address := range []sdk.AccAddress{primary} {
				delegate(t, bApp, ctx, server, denom, address, validator, math.NewInt(2000000))
				_, err := server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), validator.String(), sdk.NewInt64Coin(denom, 200000)))
				require.NoError(t, err)
				if tc.held {
					pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, address, validator)
					require.NoError(t, err)
					require.NoError(t, bApp.StakingKeeper.PutUnbondingOnHold(ctx, pending.Entries[0].UnbondingId))
				}
			}
			require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
			vault := primary
			pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, vault, validator)
			require.NoError(t, err)
			require.Len(t, pending.Entries, 1)
			if tc.prior != "" {
				_, err := bApp.StakingKeeper.SlashUnbondingDelegation(ctx, pending, pending.Entries[0].CreationHeight, math.LegacyMustNewDecFromStr(tc.prior))
				require.NoError(t, err)
				pending, err = bApp.StakingKeeper.GetUnbondingDelegation(ctx, vault, validator)
				require.NoError(t, err)
			}
			if tc.mature {
				ctx = ctx.WithBlockTime(pending.Entries[0].CompletionTime)
			}
			infraction := pending.Entries[0].CreationHeight
			if tc.tooRecent {
				infraction++
			}
			beforeSupply := bApp.BankKeeper.GetSupply(ctx, denom).Amount
			beforeLiquid := bApp.BankKeeper.GetAllBalances(ctx, vault)
			expected := pending
			expected.Entries = append([]stakingtypes.UnbondingDelegationEntry(nil), pending.Entries...)
			slashed, err := bApp.StakingKeeper.SlashUnbondingDelegation(ctx, pending, infraction, math.LegacyMustNewDecFromStr(tc.factor))
			require.NoError(t, err)
			require.Equal(t, tc.nominal, slashed.Int64())
			require.Equal(t, beforeSupply.SubRaw(tc.burn), bApp.BankKeeper.GetSupply(ctx, denom).Amount)
			require.Equal(t, beforeLiquid, bApp.BankKeeper.GetAllBalances(ctx, vault))
			// Reload the pre-slash entry metadata from its operation-ID index; the
			// slash must preserve every index and change balances only.
			for _, entry := range pending.Entries {
				indexed, err := bApp.StakingKeeper.GetUnbondingDelegationByUnbondingID(ctx, entry.UnbondingId)
				require.NoError(t, err)
				require.Equal(t, vault.String(), indexed.DelegatorAddress)
				require.Len(t, indexed.Entries, 1)
			}
			after, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, vault, validator)
			require.NoError(t, err)
			for i := range expected.Entries {
				expected.Entries[i].Balance = expected.Entries[i].Balance.SubRaw(tc.burn)
			}
			expectedBytes, err := expected.Marshal()
			require.NoError(t, err)
			afterBytes, err := after.Marshal()
			require.NoError(t, err)
			require.Equal(t, expectedBytes, afterBytes)
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
}
