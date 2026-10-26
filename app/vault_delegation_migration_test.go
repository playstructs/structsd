package app_test
import (
 "testing"
 sdk "github.com/cosmos/cosmos-sdk/types"
 stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
 stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
 "github.com/stretchr/testify/require"
 "structs/app/upgrades"
 v22 "structs/app/upgrades/v0_22_0"
)
func TestVaultDelegationMigrationRetainsAccessibleSelfDelegation(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	validator := createRedelegationValidator(t, bApp, ctx, server, denom, "vaultselfguard")
	source := sdk.AccAddress(validator)
	require.NotZero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, source.String()))
	identity := bApp.AccountKeeper.GetAccount(ctx, source)
	playerIndex := bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, source.String())
	before, err := bApp.StakingKeeper.GetDelegation(ctx, source, validator)
	require.NoError(t, err)
	keepers := &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper, StakingKeeper: bApp.StakingKeeper, AccountKeeper: bApp.AccountKeeper, DistrKeeper: bApp.DistrKeeper}
	require.NoError(t, v22.MigrateProvisionVaults(ctx, keepers))
	require.NoError(t, v22.MigrateVaultAssets(ctx, keepers))
	after, err := bApp.StakingKeeper.GetDelegation(ctx, source, validator)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, identity, bApp.AccountKeeper.GetAccount(ctx, source))
	require.Equal(t, playerIndex, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, source.String()))
	_, err = bApp.DistrKeeper.WithdrawDelegationRewards(ctx, source, validator)
	require.NoError(t, err, "retained distribution identity must remain usable")
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(source.String(), validator.String(), sdk.NewInt64Coin(denom, 1)))
	require.NoError(t, err, "native operator stake must remain accessible")
}

