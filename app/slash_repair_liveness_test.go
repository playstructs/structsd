package app_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
	"testing"
)

func TestSlashCannotBeVetoedByPendingInfusionRepair(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	target, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
	require.NoError(t, err)
	address, _, _ := newTransferPlayer(t, bApp, ctx)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, address, source, math.NewInt(2000000))
	_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(address.String(), source.String(), target.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err)

	reactor := jailGateReactorId(t, bApp, ctx, target)
	row, found := bApp.StructsKeeper.GetInfusion(ctx, reactor, address.String())
	require.True(t, found)
	row.DestinationType = types.ObjectType(500)
	bApp.StructsKeeper.SetInfusion(ctx, row)
	bApp.StructsKeeper.EnqueueReactorRestart(ctx, reactor)
	// Auxiliary repair failures do not veto native staking; repair remains queued.
	refusedCtx, _ := ctx.CacheContext()
	_, err = server.Undelegate(refusedCtx, stakingtypes.NewMsgUndelegate(address.String(), target.String(), sdk.NewInt64Coin(denom, 1)))
	require.NoError(t, err)
	before, err := bApp.StakingKeeper.GetDelegation(ctx, address, target)
	require.NoError(t, err)
	red, err := bApp.StakingKeeper.GetRedelegation(ctx, address, source, target)
	require.NoError(t, err)
	val, err := bApp.StakingKeeper.GetValidator(ctx, source)
	require.NoError(t, err)
	slashed, err := bApp.StakingKeeper.SlashRedelegation(ctx, val, red, red.Entries[0].CreationHeight, math.LegacyNewDecWithPrec(1, 1))
	require.NoError(t, err)
	require.True(t, slashed.IsPositive())
	after, err := bApp.StakingKeeper.GetDelegation(ctx, address, target)
	require.NoError(t, err)
	require.True(t, after.Shares.LT(before.Shares), "native stake must remain slashable")
	require.NotEmpty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx), "failed auxiliary repair remains queued")
}
