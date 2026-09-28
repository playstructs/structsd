package app_test

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Live gameplay notices an outage immediately. Ordinary payments still accrue
// by elapsed blocks until GridCascade removes load or the agreement expires.
func TestAgreementLazyRevenueUntilQueuedReactorCascade(t *testing.T) {
	for _, transition := range []string{"healthy", "outage", "recovered", "backed", "outage-expiry", "backed-expiry"} {
		t.Run(transition, func(t *testing.T) { testAgreementReactorOutageRevenue(t, transition) })
	}
}

func testAgreementReactorOutageRevenue(t *testing.T, transition string) {
    testAgreementReactorOutageAccounting(t, transition)
}

func testAgreementReactorOutageAccounting(t *testing.T, transition string) {
	t.Helper()
	independentlyBacked := strings.HasPrefix(transition, "backed")
	expiresBeforeCleanup := strings.HasSuffix(transition, "-expiry")
	duration := uint64(50)
	if expiresBeforeCleanup {
		duration = 2
	}
	collateral := math.NewIntFromUint64(duration).MulRaw(100000)
	// Use default v0.22 genesis and ordinary agreement messages.
	bApp, ctx := setupReactorAppGenesis(t, 2, true)
	ctx = ctx.WithBlockHeight(1)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	consensus, err := validators[0].GetConsAddr()
	require.NoError(t, err)
	reactorID := jailGateReactorId(t, bApp, ctx, validator)


	stakeDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	const denom = "ualpha"
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	// Native delegations before the provider's position make its cleanup fall
	// outside the first block's budget. No fabricated queue or infusion rows.
	for i := 0; i < types.ReactorSlashReconcileBudget; i++ {
		address := make([]byte, 20)
		address[0] = 0x69
		binary.BigEndian.PutUint64(address[12:], uint64(i+1))
		delegate(t, bApp, ctx, staking, stakeDenom, sdk.AccAddress(address), validator, math.NewInt(1000))
	}
	primary, _, ownerID := newTransferPlayer(t, bApp, ctx)
	consumerAddress, _, consumerID := newTransferPlayer(t, bApp, ctx)
	delegate(t, bApp, ctx, staking, stakeDenom, primary, validator, math.NewInt(1000000))
	if independentlyBacked {
		otherValidator, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
		require.NoError(t, err)
		delegate(t, bApp, ctx, staking, stakeDenom, primary, otherValidator, math.NewInt(1000000))
	}
	const sourceID, destinationID = "4-99100", "4-99101"
	bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: sourceID, Owner: ownerID, Creator: primary.String()})
	bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: destinationID, Owner: consumerID, Creator: consumerAddress.String()})
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	_, err = cc.NewAllocation(types.AllocationType_dynamic, ownerID, sourceID, primary.String(), ownerID, 100000)
	require.NoError(t, err)
	provider := cc.NewProvider(types.Provider{
		Owner: ownerID, Creator: primary.String(), SubstationId: sourceID,
		Rate: sdk.NewInt64Coin(denom, 2), AccessPolicy: types.ProviderAccessPolicy_openMarket,
		CapacityMinimum: 1, CapacityMaximum: 100000, DurationMinimum: 1, DurationMaximum: 100,
		ProviderCancellationPenalty: math.LegacyNewDecWithPrec(5, 1), ConsumerCancellationPenalty: math.LegacyZeroDec(),
	})
	cc.CommitAll()
	consumerVaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(consumerID)
	require.NoError(t, err)
	ownerVaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(ownerID)
	require.NoError(t, err)
	fundRedelegationAccount(t, bApp, ctx, consumerVaults[0], sdk.NewCoins(sdk.NewCoin(denom, collateral)))
	server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = server.AgreementOpen(ctx, &types.MsgAgreementOpen{Creator: consumerAddress.String(), ProviderId: provider.ID(), Capacity: 50000, Duration: duration})
	require.NoError(t, err)
	agreements := bApp.StructsKeeper.GetAllAgreementIdByProviderIndex(ctx, provider.ID())
	require.Len(t, agreements, 1)
	agreement, found := bApp.StructsKeeper.GetAgreement(ctx, agreements[0])
	require.True(t, found)
	cc = bApp.StructsKeeper.NewCurrentContext(ctx)
	allocation, found := cc.GetAllocation(agreement.AllocationId)
	require.True(t, found)
	require.NoError(t, allocation.SetDestination(destinationID))
	cc.GetPlayer(consumerID).MigrateSubstation(destinationID)
	cc.CommitAll()
	require.True(t, bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(consumerID).IsOnline())
	startTime := ctx.BlockTime()
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: startTime})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	// Stage the native departure at block 2 against the committed store, as in
	// the real-hook restoration test. The following EndBlock is the actual app.
	ctx = bApp.NewUncachedContext(false, ctx.BlockHeader()).WithBlockHeight(2).WithBlockTime(startTime.Add(time.Second))
	if transition != "healthy" {
		require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
		_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
		require.NoError(t, err)
		require.True(t, bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(consumerID).IsOnline(), "grid accounting changes when queued cleanup arrives")
	}
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: ctx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	ctx = bApp.NewUncachedContext(false, ctx.BlockHeader()).WithBlockHeight(3).WithBlockTime(startTime.Add(2 * time.Second))
	row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
	require.True(t, found)
	require.Positive(t, row.Power, "this position must still be waiting for cleanup")
	if transition != "healthy" {
		require.NotEmpty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	}
	if transition == "recovered" {
		require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
		_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
		require.NoError(t, err)
	}
	cc = bApp.StructsKeeper.NewCurrentContext(ctx)
	require.True(t, cc.GetPlayer(consumerID).IsOnline(), "the upstream position is still waiting for cleanup")
	require.Equal(t, uint64(50000), cc.GetProvider(provider.ID()).GetAgreementLoad())
	before := bApp.BankKeeper.GetBalance(ctx, ownerVaults[0], denom).Amount
	_, err = server.ProviderWithdrawBalance(ctx, &types.MsgProviderWithdrawBalance{Creator: primary.String(), ProviderId: provider.ID()})
	require.NoError(t, err)
	withdrawn := bApp.BankKeeper.GetBalance(ctx, ownerVaults[0], denom).Amount.Sub(before)
	// The checkpoint prices [1,3) from current load. Cleanup delay does not
	// create a retrospective outage credit or pause the provider clock.
	expected := int64(100000)
	message, broken := keeperlib.ProviderCollateralSolvencyInvariant(bApp.StructsKeeper)(ctx)
	require.False(t, broken, message)
	assert.Equal(t, math.NewInt(expected).String(), withdrawn.String(), "ordinary checkpoint earnings accrue until load is removed")
	if transition != "outage" && !expiresBeforeCleanup {
		return
	}
	// Continue the actual block pipeline through infusion cleanup and downstream
	// cascade settlement, using the same elapsed-block clock as withdrawals.
	for height := int64(3); height <= 8; height++ {
		blockTime := startTime.Add(time.Duration(height-1) * time.Second)
		_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: blockTime})
		require.NoError(t, err)
		_, err = bApp.Commit()
		require.NoError(t, err)
		current := bApp.NewContext(true).WithBlockHeight(height).WithBlockTime(blockTime)
		if _, exists := bApp.StructsKeeper.GetAgreement(current, agreement.Id); exists {
			continue
		}
		refund := bApp.BankKeeper.GetBalance(current, consumerVaults[0], denom).Amount
		t.Logf("settled_at=%d withdrawn_before_cleanup=%s consumer_refund=%s", height, withdrawn, refund)
		// Cascade cancellation refunds unused blocks and the forfeited holdback.
		// Expiry has no refund, even if supply was waiting in a cleanup queue.
		expectedRefund := math.ZeroInt()
		if !expiresBeforeCleanup {
			expectedRefund = collateral.SubRaw((height-1)*50000)
		}
		assert.Equal(t, expectedRefund.String(), refund.String())
		if expiresBeforeCleanup {
			require.Equal(t, int64(3), height)
			_, err = server.ProviderWithdrawBalance(current, &types.MsgProviderWithdrawBalance{Creator: primary.String(), ProviderId: provider.ID()})
			require.NoError(t, err)
			totalRevenue := bApp.BankKeeper.GetBalance(current, ownerVaults[0], denom).Amount.Sub(before)
			assert.Equal(t, collateral.String(), totalRevenue.String())
		}
		require.Zero(t, bApp.StructsKeeper.NewCurrentContext(current).GetProvider(provider.ID()).GetAgreementLoad())
		message, broken := keeperlib.ProviderCollateralSolvencyInvariant(bApp.StructsKeeper)(current)
		require.False(t, broken, message)
		return
	}
	t.Fatal("bounded cleanup did not settle this shallow fixture within six continuation blocks")
}
