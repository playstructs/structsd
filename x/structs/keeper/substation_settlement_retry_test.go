package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestAllocationSettlementFailureCanRetryInSameContext(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0")
	agreement, collateral := f.openAgreement(t, 100, 100)
	sink := sdk.AccAddress("settlement_retry_sink")
	require.NoError(t, f.k.BankKeeper().SendCoins(f.ctx, f.collateralAcc, sink, sdk.NewCoins(sdk.NewCoin(teardownDenom, collateral))))
	cc := f.k.NewCurrentContext(f.ctx)
	allocation, found := cc.GetAllocation(agreement.AllocationId)
	require.True(t, found)
	for i := 0; i < 2; i++ {
		require.Error(t, allocation.Destroy())
		require.False(t, allocation.Deleted)
		require.False(t, cc.GetAgreement(agreement.Id).IsTearingDown())
	}
	require.Equal(t, uint64(100), f.agreementLoad())
	f.fund(t, f.collateralAcc, collateral.Int64())
	require.NoError(t, allocation.Destroy())
	cc.CommitAll()
	f.requireSettledOnce(t, agreement)
	require.Zero(t, f.agreementLoad())
	require.Equal(t, collateral, f.balance(f.consumerAcc))
	require.NoError(t, allocation.Destroy())
	cc.CommitAll()
	require.Equal(t, collateral, f.balance(f.consumerAcc))
}

func TestSubstationDrainSettlementFailureStaysQueuedUntilLaterPass(t *testing.T) {
	f := setupTeardownFixture(t, 10, "1", "0")
	agreement, collateral := f.openAgreement(t, 100, 100)
	sink := sdk.AccAddress("settlement_retry_sink")
	require.NoError(t, f.k.BankKeeper().SendCoins(f.ctx, f.collateralAcc, sink, sdk.NewCoins(sdk.NewCoin(teardownDenom, collateral))))
	station, found := f.k.GetSubstation(f.ctx, f.substationId)
	require.True(t, found)
	station.DeletionPending = true
	f.k.SetSubstation(f.ctx, station)
	require.NoError(t, f.k.AppendGridCascadeQueue(f.ctx, station.Id))
	for i := 0; i < 2; i++ {
		f.ctx = sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(sdk.UnwrapSDKContext(f.ctx).BlockHeight() + 1)
		before := f.k.GetGridCascadeQueueBatch(f.ctx, 10)
		require.Len(t, before, 1)
		cc := f.k.NewCurrentContext(f.ctx)
		cc.GridCascade()
		cc.CommitAll()
		after := f.k.GetGridCascadeQueueBatch(f.ctx, 10)
		require.Len(t, after, 1)
		require.Equal(t, before[0].Sequence+1, after[0].Sequence, "one failed attempt per cascade call")
		require.Equal(t, station.Id, after[0].ObjectId)
		_, found = f.k.GetAllocation(f.ctx, agreement.AllocationId)
		require.True(t, found, "a failed refund cannot destroy the backing allocation")
		_, found = f.k.GetAgreement(f.ctx, agreement.Id)
		require.True(t, found)
		require.Equal(t, uint64(100), f.agreementLoad())
		require.True(t, f.balance(f.consumerAcc).IsZero())
	}
	f.fund(t, f.collateralAcc, collateral.Int64())
	cc := f.k.NewCurrentContext(f.ctx)
	cc.GridCascade()
	cc.CommitAll()
	f.requireSettledOnce(t, agreement)
	require.Zero(t, f.agreementLoad())
	require.Equal(t, collateral, f.balance(f.consumerAcc))
	_, found = f.k.GetSubstation(f.ctx, station.Id)
	require.False(t, found)
	cc = f.k.NewCurrentContext(f.ctx)
	cc.GridCascade()
	cc.CommitAll()
	require.Equal(t, collateral, f.balance(f.consumerAcc))
}
