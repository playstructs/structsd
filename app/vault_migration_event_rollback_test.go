package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestVaultMigrationFailureDiscardsEarlierAssociationAndStakeEvents(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	primary, _, playerID := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(1000000))
	before, err := bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
	require.NoError(t, err)
	liquid := sdk.NewCoins(sdk.NewInt64Coin(denom, 10))
	fundRedelegationAccount(t, bApp, ctx, primary, liquid)
	// An invalid legacy infusion fails after provisioning and liquid transfers.
	bApp.StructsKeeper.SetInfusion(ctx, types.Infusion{DestinationId: "3-999999", DestinationType: types.ObjectType_reactor, Address: primary.String(), PlayerId: playerID})
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	keepers := &upgrades.Keepers{StructsKeeper: bApp.StructsKeeper, StakingKeeper: bApp.StakingKeeper, AccountKeeper: bApp.AccountKeeper, DistrKeeper: bApp.DistrKeeper}
	require.ErrorContains(t, v22.MigrateVaultAssets(ctx, keepers), "missing reactor")
	// The composed migration must discard its own failed cache.
	require.Empty(t, ctx.EventManager().Events())
	require.Equal(t, liquid, bApp.BankKeeper.GetAllBalances(ctx, primary))
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, vault))
	_, indexed := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
	require.False(t, indexed)
	unchanged, err := bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
	require.NoError(t, err)
	require.Equal(t, before, unchanged)
	_, err = bApp.StakingKeeper.GetDelegation(ctx, vault, validator)
	require.Error(t, err)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
