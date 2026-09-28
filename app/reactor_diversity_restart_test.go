package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/x/structs/types"
)

func TestDiversityExportRestartAndBlockEventReplay(t *testing.T) {
	for _, zero := range []bool{false, true} {
		t.Run(map[bool]string{false: "preserve-height", true: "zero-height"}[zero], func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 4)
			address, _, owner := newTransferPlayer(t, bApp, ctx)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			for _, validator := range validators[:2] {
				val, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
				require.NoError(t, err)
				delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, address, val, math.NewInt(1000000))
			}
			before, err := bApp.StructsKeeper.OwnerReactorDiversity(ctx, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
			require.NoError(t, err)
			require.Positive(t, before.Stored.AppliedBonus)
			_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
			require.NoError(t, err)
			_, err = bApp.Commit()
			require.NoError(t, err)
			exported, err := bApp.ExportAppStateAndValidators(zero, nil, nil)
			require.NoError(t, err)
			restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
			require.NoError(t, err)
			now := ctx.BlockTime().Add(time.Second)
			_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: now, ConsensusParams: &exported.ConsensusParams})
			require.NoError(t, err)
			height := max(int64(1), exported.Height)
			restored := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: now})
			after, err := restarted.StructsKeeper.OwnerReactorDiversity(restored, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
			require.NoError(t, err)
			require.Equal(t, before, after, "base reconstruction must not retain and re-add the exported bonus")
			require.NoError(t, restarted.StructsKeeper.CheckReactorDiversityAccounting(restored))
			requireStakingAndDistributionHealthy(t, restarted, restored)
			block, err := restarted.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: now})
			require.NoError(t, err)
			reset, matched := 0, 0
			for _, event := range block.Events {
				if event.Type == "structs.structs.EventOwnerReactorDiversityReset" {
					reset++
				}
				if event.Type != "structs.structs.EventOwnerReactorDiversity" {
					continue
				}
				// BaseApp adds a raw mode label outside the protobuf payload.
				payload := abci.Event{Type: event.Type}
				for _, attribute := range event.Attributes {
					if attribute.Key != "mode" {
						payload.Attributes = append(payload.Attributes, attribute)
					}
				}
				decoded, err := sdk.ParseTypedEvent(payload)
				require.NoError(t, err)
				row := decoded.(*types.EventOwnerReactorDiversity).Diversity
				if row.OwnerId == owner {
					require.Equal(t, 1, reset, "reset precedes reconstructed owner rows")
					require.Equal(t, after.Stored, row)
					matched++
				}
			}
			require.Equal(t, 1, reset)
			require.Equal(t, 1, matched)
			_, err = restarted.Commit()
			require.NoError(t, err)
			block, err = restarted.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height + 1, Time: now.Add(time.Second)})
			require.NoError(t, err)
			for _, event := range block.Events {
				require.NotEqual(t, "structs.structs.EventOwnerReactorDiversityReset", event.Type, "snapshot replay is one-time")
			}
		})
	}
}

func TestDiversityFailedRepairSurvivesDatabaseReopen(t *testing.T) {
	database := dbm.NewMemDB()
	bApp, ctx := setupReactorAppGenesisWithDB(t, 2, true, database)
	address, _, owner := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	var target sdk.ValAddress
	for _, validator := range validators {
		val, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		require.NoError(t, err)
		delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, address, val, math.NewInt(1000000))
		target = val
	}
	before, err := bApp.StructsKeeper.OwnerReactorDiversity(ctx, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
	require.NoError(t, err)
	require.Positive(t, before.Stored.AppliedBonus)
	id := jailGateReactorId(t, bApp, ctx, target)
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	cc.GetInfusion(id, address.String()).SetRatio(0)
	cc.CommitAll()
	original, found := bApp.StructsKeeper.GetInfusion(ctx, id, address.String())
	require.True(t, found)
	broken := original
	broken.DestinationType = types.ObjectType(500)
	bApp.StructsKeeper.SetInfusion(ctx, broken)
	// A failed auxiliary hook must survive a committed block and a new app
	// instance. The malformed row forces both the main pass and retry to fail.
	require.NoError(t, bApp.StructsKeeper.Hooks().AfterDelegationModified(ctx, address, target))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	committed := bApp.NewContext(true).WithBlockTime(ctx.BlockTime())
	pending := bApp.StructsKeeper.GetReactorSlashReconcileQueue(committed)
	require.Len(t, pending, 1)
	snapshot, found, err := bApp.StructsKeeper.GetReactorRepairSnapshot(committed, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, snapshot.Through, snapshot.After, "only the failed retry remains")

	// Reuse the committed database, not exported genesis, which deliberately
	// completes repairs by rebuilding every position from native staking.
	reopened, err := app.New(log.NewNopLogger(), database, nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
	require.NoError(t, err)
	now := ctx.BlockTime().Add(time.Second)
	restored := reopened.NewContextLegacy(true, cmtproto.Header{Height: 1, ChainID: jailGateChainID, Time: now})
	require.Equal(t, pending, reopened.StructsKeeper.GetReactorSlashReconcileQueue(restored))
	afterSnapshot, found, err := reopened.StructsKeeper.GetReactorRepairSnapshot(restored, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, snapshot, afterSnapshot)
	// Repair the injected fault in the committed store branch used by the next
	// block. No state from the original app's object caches is reused.
	reopened.StructsKeeper.SetInfusion(reopened.NewUncachedContext(false, cmtproto.Header{Height: 2, ChainID: jailGateChainID, Time: now}), original)
	_, err = reopened.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: now})
	require.NoError(t, err)
	_, err = reopened.Commit()
	require.NoError(t, err)
	repaired := reopened.NewContext(true).WithBlockTime(now)
	require.Empty(t, reopened.StructsKeeper.GetReactorSlashReconcileQueue(repaired))
	after, err := reopened.StructsKeeper.OwnerReactorDiversity(repaired, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
	require.NoError(t, err)
	require.Equal(t, before, after, "retry restores the complete portfolio exactly once")
	require.NoError(t, reopened.StructsKeeper.CheckReactorDiversityAccounting(repaired))
	requireStakingAndDistributionHealthy(t, reopened, repaired)
	_, err = reopened.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 3, Time: now.Add(time.Second)})
	require.NoError(t, err)
	_, err = reopened.Commit()
	require.NoError(t, err)
	continued, err := reopened.StructsKeeper.OwnerReactorDiversity(reopened.NewContext(true), &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
	require.NoError(t, err)
	require.Equal(t, after, continued)
}
