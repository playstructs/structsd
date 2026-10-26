package app_test

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// The normal suite covers retained staking across 16 owners. Larger diagnostic
// runs use STRUCTS_MIGRATION_SCALE_PLAYERS and STRUCTS_MIGRATION_SCALE_ADDRESSES;
// the same conservation and continuation checks apply. Setup uses real SDK
// messages but no signed transactions or old
// executable. Timings and allocation counters are observations, not assertions.
func TestVaultUpgradeScaleAndMaturity(t *testing.T) {
	count := 16
	if value := os.Getenv("STRUCTS_MIGRATION_SCALE_PLAYERS"); value != "" {
		var err error
		count, err = strconv.Atoi(value)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	}
	addresses := 2
	if value := os.Getenv("STRUCTS_MIGRATION_SCALE_ADDRESSES"); value != "" {
		var err error
		addresses, err = strconv.Atoi(value)
		require.NoError(t, err)
		require.GreaterOrEqual(t, addresses, 2)
	}
	testVaultUpgradeScaleAndMaturity(t, count, addresses)
}

func TestWideVaultUpgradePreservesPendingEntriesAndLaterDefusion(t *testing.T) {
	testVaultUpgradeScaleAndMaturity(t, 2, 16)
}

func testVaultUpgradeScaleAndMaturity(t *testing.T, count, addresses int) {
	t.Helper()
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	destination, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	type ownerRecord struct {
		id            string
		sources       []sdk.AccAddress
		vault         sdk.AccAddress
		unbondings    []stakingtypes.UnbondingDelegationEntry
		redelegations []stakingtypes.RedelegationEntry
		shares        [2]math.LegacyDec
	}
	owners := make([]ownerRecord, 0, count)
	var maturity time.Time
	for i := 0; i < count; i++ {
		primary, secondary, id := newTransferPlayer(t, bApp, ctx)
		vault, err := types.VaultAddress(id)
		require.NoError(t, err)
		owner := ownerRecord{id: id, sources: []sdk.AccAddress{primary, secondary}, vault: vault}
		index := bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, primary.String())
		for len(owner.sources) < addresses {
			address := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
			require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, address.String(), index))
			owner.sources = append(owner.sources, address)
		}
		owner.shares = [2]math.LegacyDec{math.LegacyZeroDec(), math.LegacyZeroDec()}
		for j, address := range owner.sources {
			delegate(t, bApp, ctx, server, denom, address, source, math.NewInt(int64(3000000+j*1000000)))
			_, err := server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(address.String(), source.String(), destination.String(), sdk.NewInt64Coin(denom, 1000000)))
			require.NoError(t, err)
			_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), source.String(), sdk.NewInt64Coin(denom, 100000)))
			require.NoError(t, err)
			ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, address, source)
			require.NoError(t, err)
			red, err := bApp.StakingKeeper.GetRedelegation(ctx, address, source, destination)
			require.NoError(t, err)
			owner.unbondings = append(owner.unbondings, ubd.Entries...)
			owner.redelegations = append(owner.redelegations, red.Entries...)
			if maturity.IsZero() {
				maturity = ubd.Entries[0].CompletionTime
			}
			require.Equal(t, maturity, ubd.Entries[0].CompletionTime)
			require.Equal(t, maturity, red.Entries[0].CompletionTime)
			fundRedelegationAccount(t, bApp, ctx, address, sdk.NewCoins(sdk.NewInt64Coin("ualpha", int64(100*(j+1))), sdk.NewInt64Coin("oretest", int64(777+111*j))))
			for index, validator := range []sdk.ValAddress{source, destination} {
				delegation, err := bApp.StakingKeeper.GetDelegation(ctx, address, validator)
				require.NoError(t, err)
				owner.shares[index] = owner.shares[index].Add(delegation.Shares)
			}
		}
		owners = append(owners, owner)
	}
	wantLiquid := sdk.NewCoins(sdk.NewInt64Coin("ualpha", int64(100*addresses*(addresses+1)/2)), sdk.NewInt64Coin("oretest", int64(777*addresses+111*addresses*(addresses-1)/2)))
	beforeSupply := bApp.BankKeeper.GetSupply(ctx, denom)
	const upgradeName = "test-vault-scale"
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: upgradeName, Height: 2}))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	var handlerGas uint64
	var handlerElapsed time.Duration
	var allocatedBytes uint64
	bApp.UpgradeKeeper.SetUpgradeHandler(upgradeName, func(goCtx context.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
		// The fixture was written with the current code. Convert only its queue
		// layout back to the old shape outside the measured production handler.
		restoreLegacyCascadeQueueFixture(t, bApp, sdk.UnwrapSDKContext(goCtx))
		meter := storetypes.NewInfiniteGasMeter()
		measuredCtx := sdk.UnwrapSDKContext(goCtx).WithGasMeter(meter)
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		started := time.Now()
		result, err := v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(measuredCtx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
		handlerElapsed = time.Since(started)
		runtime.ReadMemStats(&after)
		allocatedBytes = after.TotalAlloc - before.TotalAlloc
		handlerGas = meter.GasConsumed()
		return result, err
	})
	runtime.GC()
	upgradeTime := ctx.BlockTime().Add(time.Second)
	started := time.Now()
	block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: upgradeTime})
	finalizeElapsed := time.Since(started)
	require.NoError(t, err)
	encoded, err := block.Marshal()
	require.NoError(t, err)
	rpc, err := cmtjson.Marshal(&coretypes.ResultBlockResults{Height: 2, TxsResults: block.TxResults, FinalizeBlockEvents: block.Events, ValidatorUpdates: block.ValidatorUpdates, ConsensusParamUpdates: block.ConsensusParamUpdates, AppHash: block.AppHash})
	require.NoError(t, err)
	started = time.Now()
	_, err = bApp.Commit()
	commitElapsed := time.Since(started)
	require.NoError(t, err)
	current := bApp.NewContextLegacy(true, cmtproto.Header{Height: 2, ChainID: jailGateChainID, Time: upgradeTime})

	require.Equal(t, beforeSupply, bApp.BankKeeper.GetSupply(current, denom))
	ubdQueue, err := bApp.StakingKeeper.GetUBDQueueTimeSlice(current, maturity)
	require.NoError(t, err)
	redQueue, err := bApp.StakingKeeper.GetRedelegationQueueTimeSlice(current, maturity)
	require.NoError(t, err)
	require.Len(t, ubdQueue, count*addresses)
	require.Len(t, redQueue, count*addresses)
	// The migration adds the vault's maturity key and leaves the original source
	// address keys. Immediate rechecks from reductions and relocation can also
	// remain queued when the fixture exceeds a block's budget. Count all raw
	// work, including keys that no longer name backing, in drain estimates.
	queueRows := bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(current)
	maturityRows, atMaturity := len(queueRows), 0
	for _, row := range queueRows {
		timestamp, _, found := strings.Cut(row, "/")
		require.True(t, found)
		when, err := sdk.ParseTimeBytes([]byte(timestamp))
		require.NoError(t, err)
		require.False(t, when.After(maturity))
		if when.Equal(maturity) {
			atMaturity++
		}
	}
	require.Equal(t, count*addresses, atMaturity)
	playerCount := bApp.StructsKeeper.GetPlayerCount(current)
	associations := map[string]int{}
	moves := map[string]int{}
	for i, event := range block.Events {
		if event.Type == "structs.structs.EventVaultAddress" {
			parsed, err := sdk.ParseTypedEvent(event)
			require.NoError(t, err)
			detail := parsed.(*types.EventVaultAddress).EventVaultAddressDetail
			_, duplicate := associations[detail.OwnerId]
			require.False(t, duplicate)
			associations[detail.OwnerId] = i
		}
		if event.Type == "structs.structs.EventVaultDelegationMigration" {
			parsed, err := sdk.ParseTypedEvent(event)
			require.NoError(t, err)
			detail := parsed.(*types.EventVaultDelegationMigration)
			association, found := associations[detail.OwnerId]
			require.True(t, found)
			require.Less(t, association, i)
			moves[detail.OwnerId]++
		}
	}
	for _, owner := range owners {
		require.Zero(t, moves[owner.id], "no staking relocation events")
		require.Equal(t, wantLiquid, bApp.BankKeeper.GetAllBalances(current, owner.vault))
		require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(current, owner.vault.String()))
		for j, address := range owner.sources {
			require.True(t, bApp.BankKeeper.GetAllBalances(current, address).IsZero())
			for i, validator := range []sdk.ValAddress{source, destination} {
				delegation, err := bApp.StakingKeeper.GetDelegation(current, address, validator)
				require.NoError(t, err)
				want := int64(1900000 + 1000000*j)
				if i == 1 { want = 1000000 }
				v, err := bApp.StakingKeeper.GetValidator(current, validator)
				require.NoError(t, err)
				require.Equal(t, want, v.TokensFromShares(delegation.Shares).TruncateInt64())
				infusion, found := bApp.StructsKeeper.GetInfusion(current, jailGateReactorId(t, bApp, current, validator), address.String())
				require.True(t, found)
				require.Equal(t, owner.id, infusion.PlayerId)
				require.Equal(t, uint64(want), infusion.Fuel)
				_, err = bApp.StakingKeeper.GetDelegation(current, owner.vault, validator)
				require.ErrorIs(t, err, stakingtypes.ErrNoDelegation)
			}
			ubd, err := bApp.StakingKeeper.GetUnbondingDelegationByUnbondingID(current, owner.unbondings[j].UnbondingId)
			require.NoError(t, err)
			require.Equal(t, address.String(), ubd.DelegatorAddress)
			require.Equal(t, []stakingtypes.UnbondingDelegationEntry{owner.unbondings[j]}, ubd.Entries)
			red, err := bApp.StakingKeeper.GetRedelegationByUnbondingID(current, owner.redelegations[j].UnbondingId)
			require.NoError(t, err)
			require.Equal(t, address.String(), red.DelegatorAddress)
			require.Equal(t, []stakingtypes.RedelegationEntry{owner.redelegations[j]}, red.Entries)
		}
	}
	requireStakingAndDistributionHealthy(t, bApp, current)
	t.Logf("UPGRADE players=%d addresses=%d pending_entries=%d handler_gas=%d handler_elapsed=%s allocated_bytes=%d finalize_elapsed=%s commit_elapsed=%s events=%d abci_bytes=%d rpc_result_bytes=%d", count, count*addresses, count*addresses*2, handlerGas, handlerElapsed, allocatedBytes, finalizeElapsed, commitElapsed, len(block.Events), len(encoded), len(rpc))
	defuse := &types.MsgReactorDefuse{Creator: owners[0].sources[0].String(), DelegatorAddress: owners[0].sources[0].String(), ValidatorAddress: source.String(), Amount: sdk.NewInt64Coin(denom, 1)}
	// Address count no longer consumes one shared pending-entry limit.
	preview, _ := current.CacheContext()
	_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).ReactorDefuse(preview, defuse)
	require.NoError(t, err)

	// Native staking completes every due refund in its own EndBlock. Structs
	// bookkeeping can take more blocks, without delaying or repeating a credit.
	for pass := 0; pass < 2+maturityRows/types.InfusionMaturitySweepBudget; pass++ {
		blockTime := maturity.Add(time.Duration(pass+1) * time.Second)
		started := time.Now()
		result, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: int64(3 + pass), Time: blockTime})
		elapsed := time.Since(started)
		require.NoError(t, err)
		_, err = bApp.Commit()
		require.NoError(t, err)
		current = bApp.NewContextLegacy(true, cmtproto.Header{Height: int64(3 + pass), ChainID: jailGateChainID, Time: blockTime})
		for _, owner := range owners {
			require.Equal(t, wantLiquid, bApp.BankKeeper.GetAllBalances(current, owner.vault))
			for _, address := range owner.sources {
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 100000)), bApp.BankKeeper.GetAllBalances(current, address))
			}
		}
		require.Len(t, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(current), max(0, maturityRows-(pass+1)*types.InfusionMaturitySweepBudget))
		require.Equal(t, playerCount, bApp.StructsKeeper.GetPlayerCount(current), "stale source queue rows must not register players")
		if pass == 0 {
			t.Logf("MATURITY players=%d raw_queue_rows=%d finalize_elapsed=%s events=%d", count, maturityRows, elapsed, len(result.Events))
		}
	}
	for _, owner := range owners {
		for _, address := range owner.sources {
			_, err := bApp.StakingKeeper.GetUnbondingDelegation(current, address, source)
			require.ErrorIs(t, err, stakingtypes.ErrNoUnbondingDelegation)
			_, err = bApp.StakingKeeper.GetRedelegation(current, address, source, destination)
			require.ErrorIs(t, err, stakingtypes.ErrNoRedelegation)
			infusion, found := bApp.StructsKeeper.GetInfusion(current, jailGateReactorId(t, bApp, current, source), address.String())
			require.True(t, found)
			require.Zero(t, infusion.Defusing)
		}
	}
	require.Empty(t, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(current))
	requireStakingAndDistributionHealthy(t, bApp, current)
	// Once pending operations finish, the player's signing key can defuse its
	// legacy stake through the real Structs handler.
	cache, _ := current.CacheContext()
	meter := storetypes.NewGasMeter(20000000)
	cache = cache.WithGasMeter(meter)
	response, err := keeperlib.NewMsgServerImpl(bApp.StructsKeeper).ReactorDefuse(cache, defuse)
	require.NoError(t, err)
	require.Equal(t, defuse.Amount, response.Amount)
	row, err := bApp.StakingKeeper.GetUnbondingDelegation(cache, owners[0].sources[0], source)
	require.NoError(t, err)
	require.Len(t, row.Entries, 1)
	require.Equal(t, int64(1), row.Entries[0].Balance.Int64())
	t.Logf("POST_MATURITY_DEFUSE addresses_per_owner=%d gas=%d", addresses, meter.GasConsumed())
}
