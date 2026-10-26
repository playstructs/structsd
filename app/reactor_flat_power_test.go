package app_test

import (
	"bytes"
	"sort"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestFlatReactorStakingHooksPreserveBacking(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 5)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	sort.Slice(validators, func(i, j int) bool {
		a, _ := sdk.ValAddressFromBech32(validators[i].OperatorAddress)
		b, _ := sdk.ValAddressFromBech32(validators[j].OperatorAddress)
		return bytes.Compare(a, b) < 0
	})
	validator, err := sdk.ValAddressFromBech32(validators[4].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(1000000))
	reactorBytes, found := bApp.StructsKeeper.GetReactorBytesFromValidator(ctx, validator)
	require.True(t, found)
	reactor, found := bApp.StructsKeeper.GetReactorByBytes(ctx, reactorBytes)
	require.True(t, found)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(1000000))
	toppedUp, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
	require.True(t, found)
	require.Equal(t, uint64(2000000), toppedUp.Fuel)
	require.Equal(t, uint64(2000000), toppedUp.Power)

	require.NoError(t, toppedUp.ValidateAssetOwner())
	// Exercise existing gate/restore methods against real staking jail state.
	val, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	consensus, err := val.GetConsAddr()
	require.NoError(t, err)
	require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
	bApp.StructsKeeper.ReactorGateEnergy(ctx, validator)
	gated, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
	require.True(t, found)
	require.Zero(t, gated.Power)

	require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
	bApp.StructsKeeper.ReactorRestoreEnergy(ctx, validator)
	restored, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
	require.True(t, found)
	require.Equal(t, toppedUp, restored)
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err)
	partial, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
	require.True(t, found)
	require.Equal(t, uint64(1000000), partial.Fuel)
	require.Equal(t, uint64(1000000), partial.Power)

	require.Equal(t, uint64(1000000), partial.Defusing)
	// Distinct creation heights produce distinct entries and hook callbacks.
	// Same-block SDK entry merging omits the second unbonding hook entirely.
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err)
	empty, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
	require.True(t, found)
	require.Zero(t, empty.Fuel)
	require.Zero(t, empty.Power)

	require.Equal(t, uint64(2000000), empty.Defusing)
	require.NoError(t, empty.ValidateAssetOwner())
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
