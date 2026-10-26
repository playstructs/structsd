package app_test

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Exercise the configured EndBlock module order with overlapping ordinary
// settlement, cascade and generator work. The staking-overlap case also runs
// BeginBlock and native maturity plus slash repair. Gas and sizes are diagnostics,
// not transaction limits or a full ABCI/Commit performance claim. Fixture setup,
// inspections and serialization are outside the measured module call.
func TestOrdinaryEndBlockCombinedWork(t *testing.T) {
	for _, scenario := range []struct {
		name               string
		agreements, grants int
		stakingWork        bool
	}{
		{"admission-cap", types.AgreementExpirationBucketCap, 0, false},
		{"admission-cap-with-grants", types.AgreementExpirationBucketCap, types.PermissionCleanupBudget + 1, false},
		{"legacy-overfull", types.AgreementExpirationBucketCap * 2, 0, false},
		{"staking-overlap", types.AgreementExpirationBucketCap, types.PermissionCleanupBudget + 1, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			bApp, ctx, generatorID, _, donors := setupGeneratorQueueApp(t, types.GeneratorPowerReconcileBudget+2)
			_, _, providerID, ids, consumerVaults := setupFundedAgreementsOnApp(t, bApp, ctx, 1)
			k := bApp.StructsKeeper
			template, found := k.GetAgreement(ctx, ids[0])
			require.True(t, found)
			allocation, found := k.GetAllocation(ctx, template.AllocationId)
			require.True(t, found)
			cc := k.NewCurrentContext(ctx)
			// Import an internally consistent legacy bucket. Opening new agreements
			// cannot construct the overfull case after the admission cap activates.
			for i := 1; i < scenario.agreements; i++ {
				row := template
				row.Id = keeperlib.GetObjectID(types.ObjectType_agreement, uint64(800000+i))
				row.AllocationId = keeperlib.GetObjectID(types.ObjectType_allocation, uint64(800000+i))
				link := allocation
				link.Id, link.Index = row.AllocationId, uint64(800000+i)
				cc.GenesisImportAllocation(link, row.Capacity)
				cc.GenesisImportAgreement(row)
				ids = append(ids, row.Id)
			}
			cc.CommitAll()
			fundRedelegationAccount(t, bApp, ctx, keeperlib.GetProviderCollateralPoolLocation(providerID), sdk.NewCoins(sdk.NewInt64Coin("ualpha", int64(21*(scenario.agreements-1)))))
			require.Len(t, k.GetAllAgreementIdByExpirationIndex(ctx, template.EndBlock), scenario.agreements)
			require.False(t, k.AgreementExpirationHeightHasRoomFor(ctx, template.EndBlock, ""))

			// All grants name real players, shared across the agreements. Block
			// teardown uses only the shared permission cleanup worker.
			for i := 0; i < scenario.grants; i++ {
				address := make([]byte, 20)
				address[0] = 0x63
				binary.BigEndian.PutUint64(address[12:], uint64(i+1))
				player := appendAppPlayer(t, k, ctx, sdk.AccAddress(address))
				for _, id := range ids {
					k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(id, player.Id), types.PermUpdate)
				}
			}

			// A separate one-power chain keeps cascade work independent of expiry.
			// More links than the row/queue-visit budget can drain must survive
			// the first block, then make progress on subsequent blocks.
			_, _, gridOwner := newTransferPlayer(t, bApp, ctx)
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, gridOwner), 1)
			cc = k.NewCurrentContext(ctx)
			source := gridOwner
			const links = types.GridCascadeBlockBudget + 2
			for i := 0; i < links; i++ {
				next := fmt.Sprintf("4-%08d", 850000+i)
				k.SetSubstation(ctx, types.Substation{Id: next, Owner: gridOwner})
				cc.GenesisImportAllocation(types.Allocation{Id: fmt.Sprintf("6-%08d", 850000+i), SourceObjectId: source, DestinationId: next, Controller: gridOwner, Type: types.AllocationType_dynamic}, 1)
				source = next
			}
			cc.CommitAll()
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, gridOwner), 0)
			k.AppendGridCascadeQueue(ctx, gridOwner)
			cc = k.NewCurrentContext(ctx)
			cc.GetStruct(generatorID).GoOffline()
			cc.CommitAll()

			var stakingWork *ordinaryStakingWork
			if scenario.stakingWork {
				stakingWork, ctx = setupOrdinaryStakingWork(t, bApp, ctx)
			}
			ctx = ctx.WithBlockHeight(int64(template.EndBlock))
			var firstGas uint64
			var firstEvents, firstABCI, firstRPC int
			completedPasses := 0
			// Permission storage may outlive settlement. Allow enough blocks to
			// collect every row through the shared budget, checking money each
			// time so delayed cleanup cannot accidentally repeat settlement.
			maxPasses := 10 + 2*scenario.agreements*(scenario.grants+2)/types.PermissionCleanupBudget
			for pass := 0; pass < maxPasses; pass++ {
				meter := storetypes.NewInfiniteGasMeter()
				header := ctx.HeaderInfo()
				header.Height, header.Time = ctx.BlockHeight(), ctx.BlockTime()
				ctx = ctx.WithHeaderInfo(header)
				started := time.Now()
				var beginEvents []abci.Event
				if scenario.stakingWork {
					begin, err := bApp.ModuleManager.BeginBlock(ctx.WithGasMeter(meter))
					require.NoError(t, err)
					beginEvents = begin.Events
				}
				result, err := bApp.ModuleManager.EndBlock(ctx.WithGasMeter(meter))
				elapsed := time.Since(started)
				require.NoError(t, err)
				result.Events = append(beginEvents, result.Events...)
				permissionEvents := 0
				for _, event := range result.Events {
					if event.Type == "structs.structs.EventPermission" || event.Type == "structs.structs.EventGuildRankPermission" {
						permissionEvents++
					}
				}
				require.LessOrEqual(t, permissionEvents, types.PermissionCleanupBudget, "all block-hook deletions must share the cleanup budget")
				encoded, err := (&abci.ResponseFinalizeBlock{Events: result.Events, ValidatorUpdates: result.ValidatorUpdates}).Marshal()
				require.NoError(t, err)
				rpc, err := cmtjson.Marshal(&coretypes.ResultBlockResults{Height: ctx.BlockHeight(), FinalizeBlockEvents: result.Events, ValidatorUpdates: result.ValidatorUpdates})
				require.NoError(t, err)
				if pass == 0 {
					firstGas, firstEvents, firstABCI, firstRPC = meter.GasConsumed(), len(result.Events), len(encoded), len(rpc)
					require.NotZero(t, k.GetGridCascadeQueueLength(ctx))
					_, queued, err := k.GetGeneratorPowerSnapshot(ctx, generatorID)
					require.NoError(t, err)
					require.True(t, queued, "generator backlog must remain bounded despite simultaneous expiry")
					if scenario.grants > types.PermissionCleanupBudget {
						require.NotEmpty(t, k.GetPermissionCleanupQueue(ctx))
					}
				}
				// Full-term revenue follows the existing aggregate checkpoint and
				// per-agreement truncation: floor(12.6*N) + 8*N, with dust retained.
				earned := int64(126*scenario.agreements/10 + 8*scenario.agreements)
				require.Equal(t, earned, bApp.BankKeeper.GetBalance(ctx, keeperlib.GetProviderEarningsPoolLocation(providerID), "ualpha").Amount.Int64())
				require.Equal(t, int64(21*scenario.agreements)-earned, bApp.BankKeeper.GetBalance(ctx, keeperlib.GetProviderCollateralPoolLocation(providerID), "ualpha").Amount.Int64())
				require.True(t, bApp.BankKeeper.GetAllBalances(ctx, consumerVaults[0]).IsZero(), "a fully served consumer receives no refund")
				require.Zero(t, k.NewCurrentContext(ctx).GetProvider(providerID).GetAgreementLoad())
				require.Empty(t, k.GetAllAgreementIdByProviderIndex(ctx, providerID))
				require.Empty(t, k.GetAllAgreementIdByExpirationIndex(ctx, template.EndBlock))
				gridBacklog, cleanupBacklog := k.GetGridCascadeQueueLength(ctx), len(k.GetPermissionCleanupQueue(ctx))
				_, generatorQueued, err := k.GetGeneratorPowerSnapshot(ctx, generatorID)
				require.NoError(t, err)
				done := gridBacklog == 0 && cleanupBacklog == 0 && !generatorQueued
				if stakingWork != nil {
					stakingDone := stakingWork.verify(t, bApp, ctx, pass, result.Events)
					done = done && stakingDone
				}
				if pass < 4 || done {
					t.Logf("pass=%d gas=%d events=%d abci_bytes=%d rpc_result_bytes=%d elapsed=%s grid_backlog=%d cleanup_backlog=%d", pass, meter.GasConsumed(), len(result.Events), len(encoded), len(rpc), elapsed, gridBacklog, cleanupBacklog)
				}
				completedPasses++
				ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
				if done && pass >= 3 {
					break
				}
			}
			require.Zero(t, k.GetGridCascadeQueueLength(ctx))
			require.Empty(t, k.GetPermissionCleanupQueue(ctx))
			if stakingWork != nil {
				require.True(t, stakingWork.verify(t, bApp, ctx, completedPasses, nil), "all staking bookkeeping must eventually drain")
			}
			_, queued, err := k.GetGeneratorPowerSnapshot(ctx, generatorID)
			require.NoError(t, err)
			require.False(t, queued)
			for _, donor := range donors {
				require.Zero(t, k.NewCurrentContext(ctx).GetPlayer(donor.Id).GetCapacity())
			}
			for _, id := range ids {
				_, found := k.GetAgreement(ctx, id)
				require.False(t, found)
				require.Empty(t, k.GetPermissionsByObject(ctx, id))
			}
			t.Logf("SUMMARY agreements=%d direct_grants_per_agreement=%d chain_links=%d generator_donors=%d gas=%d events=%d abci_bytes=%d rpc_result_bytes=%d passes=%d", scenario.agreements, scenario.grants, links, len(donors), firstGas, firstEvents, firstABCI, firstRPC, completedPasses)
		})
	}
}
