package app_test

import (
	"encoding/binary"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestBondedReactorRestoresInBoundedBlocks(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	consensus, err := validators[0].GetConsAddr()
	require.NoError(t, err)
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	addresses := make([]sdk.AccAddress, types.ReactorSlashReconcileBudget+2)
	for i := range addresses {
		address := make([]byte, 20)
		address[0] = 0x77
		binary.BigEndian.PutUint64(address[12:], uint64(i+1))
		addresses[i] = sdk.AccAddress(address)
		delegate(t, bApp, ctx, staking, denom, addresses[i], validator, math.NewInt(1000))
	}
	require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	// Drive shutdown through actual block execution before testing recovery.
	// BeginBlock genesis replay is not a new shutdown delta; only EndBlock
	// infusion events may advertise the positions cleaned in that block.
	shutdownTargets := map[string]bool{}
	for _, address := range addresses {
		shutdownTargets[address.String()] = true
	}
	previousGated := 0
	initialTime := ctx.BlockTime()
	for height := int64(1); height <= 2; height++ {
		blockTime := initialTime.Add(time.Duration(height) * time.Second)
		block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: blockTime})
		require.NoError(t, err)
		advertised := 0
		for _, event := range block.Events {
			if event.Type != "structs.structs.EventInfusion" {
				continue
			}
			payload := abci.Event{Type: event.Type}
			mode := ""
			for _, attribute := range event.Attributes {
				if attribute.Key == "mode" {
					mode = attribute.Value
				} else {
					payload.Attributes = append(payload.Attributes, attribute)
				}
			}
			if mode != "EndBlock" {
				continue
			}
			decoded, err := sdk.ParseTypedEvent(payload)
			require.NoError(t, err)
			row := decoded.(*types.EventInfusion).Infusion
			if row.DestinationId == reactorID && shutdownTargets[row.Address] {
				require.Zero(t, row.Power)
				advertised++
			}
		}
		current := bApp.NewContext(false).WithBlockTime(blockTime)
		gated := 0
		for _, address := range addresses {
			row, found := bApp.StructsKeeper.GetInfusion(current, reactorID, address.String())
			require.True(t, found)
			require.Equal(t, uint64(1000), row.Fuel)
			if row.Power == 0 {
				gated++
			}
		}
		require.Positive(t, gated-previousGated)
		require.LessOrEqual(t, gated-previousGated, types.ReactorSlashReconcileBudget)
		require.Equal(t, gated-previousGated, advertised)
		if height == 1 {
			require.Less(t, gated, len(addresses))
		} else {
			require.Equal(t, len(addresses), gated)
		}
		_, err = bApp.Commit()
		require.NoError(t, err)
		ctx = bApp.NewContext(true).WithBlockTime(blockTime).WithBlockHeight(height + 1)
		previousGated = gated
	}
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	// Commit clears BaseApp's finalize state. Stage this test's native rebond
	// against the committed store so the next FinalizeBlock sees it; mutations
	// to CheckTx's separate cache would be discarded at the next commit.
	ctx = bApp.NewUncachedContext(false, ctx.BlockHeader())
	require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	queue := bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 1)
	require.Equal(t, reactorID, queue[0].ReactorId)
	for _, address := range addresses {
		row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, address.String())
		require.True(t, found)
		require.Zero(t, row.Power, "real SDK rebond queues work without restoring positions inline")
	}
	// Seed an indexer's last known state before restoration. Only public block
	// events may update these projections during the next two blocks.
	indexedInfusions := map[string]types.Infusion{}
	for _, row := range bApp.StructsKeeper.GetAllInfusion(ctx) {
		indexedInfusions[row.DestinationId+"-"+row.Address] = row
	}
	indexedGrid := map[string]uint64{}
	for _, row := range bApp.StructsKeeper.GetAllGridExport(ctx) {
		if row.Value != 0 {
			indexedGrid[row.AttributeId] = row.Value
		}
	}
	targets := map[string]bool{}
	for _, address := range addresses {
		targets[reactorID+"-"+address.String()] = true
	}
	previousRestored := 0
	for height := int64(3); height <= 4; height++ {
		blockTime := ctx.BlockTime().Add(time.Duration(height) * time.Second)
		block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: blockTime})
		require.NoError(t, err)
		advertisedRestorations := 0
		for _, event := range block.Events {
			switch event.Type {
			case "structs.structs.EventInfusion", "structs.structs.EventGrid", "structs.structs.EventDelete":
			default:
				continue
			}
			payload := abci.Event{Type: event.Type}
			for _, attribute := range event.Attributes {
				if attribute.Key == "mode" {
					require.Contains(t, []string{"BeginBlock", "EndBlock"}, attribute.Value)
					continue
				}
				payload.Attributes = append(payload.Attributes, attribute)
			}
			decoded, err := sdk.ParseTypedEvent(payload)
			require.NoError(t, err)
			switch detail := decoded.(type) {
			case *types.EventInfusion:
				row := *detail.Infusion
				key := row.DestinationId + "-" + row.Address
				if targets[key] && indexedInfusions[key].Power == 0 && row.Power != 0 {
					advertisedRestorations++
				}
				indexedInfusions[key] = row
			case *types.EventGrid:
				row := detail.GridRecord
				if row.Value == 0 {
					delete(indexedGrid, row.AttributeId)
				} else {
					indexedGrid[row.AttributeId] = row.Value
				}
			case *types.EventDelete:
				delete(indexedInfusions, detail.ObjectId)
			}
		}
		current := bApp.NewContext(false).WithBlockTime(blockTime)
		cc := bApp.StructsKeeper.NewCurrentContext(current)
		restored := 0
		for _, address := range addresses {
			row, found := bApp.StructsKeeper.GetInfusion(current, reactorID, address.String())
			require.True(t, found)
			require.Equal(t, uint64(1000), row.Fuel)
			require.Contains(t, []uint64{0, 1000}, row.Power)
			if row.Power != 0 {
				restored++
			}
			player, err := cc.GetPlayerByAddress(address.String())
			require.NoError(t, err)
			_, _, ownerPower := row.GetPowerDistribution()
			require.Equal(t, ownerPower, player.GetCapacity())
		}
		require.Greater(t, restored, previousRestored)
		require.LessOrEqual(t, restored-previousRestored, types.ReactorSlashReconcileBudget)
		require.Equal(t, restored-previousRestored, advertisedRestorations, "events advertise exactly this block's completed restorations")
		actualInfusions := map[string]types.Infusion{}
		for _, row := range bApp.StructsKeeper.GetAllInfusion(current) {
			actualInfusions[row.DestinationId+"-"+row.Address] = row
		}
		requireInfusionEventsEqual(t, actualInfusions, indexedInfusions, "replay must preserve all infusion fields, including pending zero-power positions")
		actualGrid := map[string]uint64{}
		for _, row := range bApp.StructsKeeper.GetAllGridExport(current) {
			if row.Value != 0 {
				actualGrid[row.AttributeId] = row.Value
			}
		}
		require.Equal(t, actualGrid, indexedGrid, "block events reconstruct effective grid state during partial recovery")
		if height == 3 {
			require.Less(t, restored, len(addresses))
			require.NotEmpty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(current))
		} else {
			require.Equal(t, len(addresses), restored)
			require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(current))
		}
		requireStakingAndDistributionHealthy(t, bApp, current)
		_, err = bApp.Commit()
		require.NoError(t, err)
		previousRestored = restored
	}
}

// Real staking hooks exercise the ordering here; the committed-block test
// above separately verifies where the bounded processor runs in FinalizeBlock.
func TestPartialRestorationSurvivesSlashAndRejail(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	consensus, err := validators[0].GetConsAddr()
	require.NoError(t, err)
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	addresses := make([]sdk.AccAddress, types.ReactorSlashReconcileBudget+2)
	for i := range addresses {
		address := make([]byte, 20)
		address[0] = 0x76
		binary.BigEndian.PutUint64(address[12:], uint64(i+1))
		addresses[i] = sdk.AccAddress(address)
		delegate(t, bApp, ctx, staking, denom, addresses[i], validator, math.NewInt(1000000))

	}

	setJailed := func(jailed bool) {
		t.Helper()
		if jailed {
			require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
		} else {
			require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
		}
		_, err := bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
		require.NoError(t, err)
	}
	setJailed(true)
	for i := 0; i < 2; i++ {
		bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	}
	setJailed(false)
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	restored := 0
	for _, address := range addresses {
		row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, address.String())
		require.True(t, found)
		if row.Power != 0 {
			restored++
		}
	}
	require.Positive(t, restored)
	require.Less(t, restored, len(addresses))
	require.NotEmpty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(),
		value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
	require.NoError(t, err)
	setJailed(true)
	checkPower := func(enabled bool, settled bool) {
		t.Helper()
		cc := bApp.StructsKeeper.NewCurrentContext(ctx)
		for _, address := range addresses {
			row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, address.String())
			require.True(t, found)
			if settled {
				require.Equal(t, uint64(950000), row.Fuel)

			}
			if enabled {
				require.Equal(t, uint64(950000), row.Power)
			} else if settled {
				require.Zero(t, row.Power, "completed shutdown must have zero stored power")
			}
			player, err := cc.GetPlayerByAddress(address.String())
			require.NoError(t, err)
			if !enabled {
				require.False(t, player.IsOnline(), "unsettled stored power cannot keep a jailed owner online")
			}
			_, _, ownerPower := row.GetPowerDistribution()
			require.Equal(t, ownerPower, player.GetCapacity())
		}
	}
	checkPower(false, false)
	for i := 0; i < 2; i++ {
		bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
		checkPower(false, i == 1)
	}
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	setJailed(false)
	checkPower(false, true)
	for i := 0; i < 2; i++ {
		bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	}
	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
	checkPower(true, true)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
