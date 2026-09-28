package app_test

import (
	"bytes"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestRetainedSelfStakePreservesNativeMinimumAndUnjailRecovery(t *testing.T) {
	for _, bonded := range []bool{false, true} {
		name := "never_bonded"
		if bonded {
			name = "bonded"
		}
		t.Run(name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			validator := createRedelegationValidator(t, bApp, ctx, server, denom, "retainedminimum")
			operator := sdk.AccAddress(validator)
			if bonded {
				_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
				require.NoError(t, err)
				active, err := bApp.StakingKeeper.GetValidator(ctx, validator)
				require.NoError(t, err)
				require.True(t, active.IsBonded())
			}
			minimum := math.NewInt(10000000)
			_, err = server.EditValidator(ctx, stakingtypes.NewMsgEditValidator(validator.String(), stakingtypes.Description{Moniker: "retainedminimum"}, nil, &minimum))
			require.NoError(t, err)
			playerIndex := bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, operator.String())
			require.NotZero(t, playerIndex)
			secondary := sdk.AccAddress(bytes.Repeat([]byte{0x49}, 20))
			require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, secondary.String(), playerIndex))
			delegate(t, bApp, ctx, server, denom, secondary, validator, math.NewInt(2000000))
			require.NoError(t, v22.MigrateVaultAssets(ctx, delegationUpgradeKeepers(bApp)))
			player, err := bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayerByAddress(operator.String())
			require.NoError(t, err)
			vault, err := types.VaultAddress(player.ID())
			require.NoError(t, err)
			_, err = bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
			require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
			vaultStake, err := bApp.StakingKeeper.GetDelegation(ctx, secondary, validator)
			require.NoError(t, err)
			require.True(t, vaultStake.Shares.IsPositive())
			_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(operator.String(), validator.String(), sdk.NewInt64Coin(denom, 1)))
			require.NoError(t, err)
			jailed, err := bApp.StakingKeeper.GetValidator(ctx, validator)
			require.NoError(t, err)
			require.True(t, jailed.Jailed, "vault stake must not count toward the retained native minimum")
			unjail := slashingkeeper.NewMsgServerImpl(bApp.SlashingKeeper)
			_, err = unjail.Unjail(ctx, &slashingtypes.MsgUnjail{ValidatorAddr: validator.String()})
			require.ErrorIs(t, err, slashingtypes.ErrSelfDelegationTooLowToUnjail)
			delegate(t, bApp, ctx, server, denom, operator, validator, math.OneInt())
			_, err = unjail.Unjail(ctx, &slashingtypes.MsgUnjail{ValidatorAddr: validator.String()})
			require.NoError(t, err, "restoring the retained self-stake must permit native unjailing")
			recovered, err := bApp.StakingKeeper.GetValidator(ctx, validator)
			require.NoError(t, err)
			require.False(t, recovered.Jailed)
			afterVaultStake, err := bApp.StakingKeeper.GetDelegation(ctx, secondary, validator)
			require.NoError(t, err)
			require.Equal(t, vaultStake, afterVaultStake)
		})
	}
}
