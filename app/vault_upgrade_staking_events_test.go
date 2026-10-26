package app_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

// Reconstruct the staking records from a pre-upgrade snapshot and only the
// public block events. Keeper event tests cannot prove PreBlock delivery, and
// merely counting events cannot detect incomplete destination records.
func TestVaultUpgradeBlockEventsReconstructStakingRecords(t *testing.T) {
	testVaultUpgradeBlockReplay(t, false, false, false)
}

func TestVaultAndPowerUpgradeBlockEventsReconstructState(t *testing.T) {
	testVaultUpgradeBlockReplay(t, true, false, false)
}

func TestVaultAndPowerUpgradeBrownoutShedsAllocation(t *testing.T) {
	testVaultUpgradeBlockReplay(t, true, true, false)
}

func TestV22HandlerUpgradeEventsReconstructState(t *testing.T) {
	testVaultUpgradeBlockReplay(t, true, false, true)
}

func TestV22HandlerUpgradeBrownoutContinues(t *testing.T) {
	testVaultUpgradeBlockReplay(t, true, true, true)
}

func testVaultUpgradeBlockReplay(t *testing.T, rebase, brownout, namedHandler bool) {
	t.Helper()
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	target, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	for _, address := range []sdk.AccAddress{primary, secondary} {
		delegate(t, bApp, ctx, server, denom, address, source, math.NewInt(3000000))
		_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), source.String(), sdk.NewInt64Coin(denom, 100000)))
		require.NoError(t, err)
		_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(address.String(), source.String(), target.String(), sdk.NewInt64Coin(denom, 200000)))
		require.NoError(t, err)
	}
	var shedAllocation string
	if brownout {
		// Model historical phantom fuel. Reconciliation must remove this bonus
		// before rebuilding legitimate stake, even if that sheds load.
		cc := bApp.StructsKeeper.NewCurrentContext(ctx)
		infusion := cc.GetInfusion(jailGateReactorId(t, bApp, ctx, source), primary.String())
		infusion.SetFuel(infusion.GetFuel() + 1000000)
		cc.CommitAll()
	}
	if brownout {
		cc := bApp.StructsKeeper.NewCurrentContext(ctx)
		player := cc.GetPlayer(playerID)
		capacity := player.GetCapacity()
		require.Positive(t, capacity)
		allocation, err := cc.NewAllocation(types.AllocationType_static, playerID, "", primary.String(), playerID, 1)
		require.NoError(t, err)
		// Historical fixtures may carry allocations that exceed current live backing.
		// Reproduce that stored state without admitting it through today's capacity gate.
		_, err = allocation.SetPower(capacity)
		require.NoError(t, err)
		shedAllocation = allocation.ID()
		cc.CommitAll()
	}
	const name = "test-vault-staking-event-replay"
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: name, Height: 2}))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	beforeCtx := bApp.NewContextLegacy(true, cmtproto.Header{Height: 1, ChainID: jailGateChainID, Time: ctx.BlockTime()})
	before := bApp.StakingKeeper.ExportGenesis(beforeCtx)
	infusions := map[string]types.Infusion{}
	for _, row := range bApp.StructsKeeper.GetAllInfusion(beforeCtx) {
		infusions[row.DestinationId+"-"+row.Address] = row
	}
	grid := map[string]uint64{}
	for _, row := range bApp.StructsKeeper.GetAllGridExport(beforeCtx) {
		if row.Value != 0 {
			grid[row.AttributeId] = row.Value
		}
	}
	delegations := map[[2]string]stakingtypes.Delegation{}
	unbondings := map[[2]string]stakingtypes.UnbondingDelegation{}
	redelegations := map[[3]string]stakingtypes.Redelegation{}
	for _, row := range before.Delegations {
		delegations[[2]string{row.DelegatorAddress, row.ValidatorAddress}] = row
	}
	for _, row := range before.UnbondingDelegations {
		unbondings[[2]string{row.DelegatorAddress, row.ValidatorAddress}] = row
	}
	for _, row := range before.Redelegations {
		redelegations[[3]string{row.DelegatorAddress, row.ValidatorSrcAddress, row.ValidatorDstAddress}] = row
	}
	bApp.UpgradeKeeper.SetUpgradeHandler(name, func(goCtx context.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
		if namedHandler {
			restoreLegacyCascadeQueueFixture(t, bApp, sdk.UnwrapSDKContext(goCtx))
			next, err := v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(goCtx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
			if brownout && err == nil {
				player := bApp.StructsKeeper.NewCurrentContext(sdk.UnwrapSDKContext(goCtx)).GetPlayer(playerID)
				require.Greater(t, player.GetLoad(), player.GetCapacity(), "named handler must leave the fixture overloaded before EndBlock")
			}
			return next, err
		}
		if err := v22.MigrateVaultAssets(goCtx, delegationUpgradeKeepers(bApp)); err != nil {
			return nil, err
		}
		if rebase {
			// Rebuild flat power after all vault backing has moved.
			if err := v22.MigrateReactorPower(goCtx, delegationUpgradeKeepers(bApp)); err != nil {
				return nil, err
			}
			if brownout {
				cc := bApp.StructsKeeper.NewCurrentContext(sdk.UnwrapSDKContext(goCtx))
				player := cc.GetPlayer(playerID)
				require.Greater(t, player.GetLoad(), player.GetCapacity(), "fixture must brown out before EndBlock")
			}
		}
		return versions, nil
	})
	blockTime := ctx.BlockTime().Add(time.Second)
	result, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: blockTime})
	require.NoError(t, err)
	require.Empty(t, result.TxResults)
	associations := map[string]string{}
	counts := map[string]int{}
	infusionEvents, gridEvents, removedInfusions := 0, 0, 0
	allocationDeleted := false
	for _, event := range result.Events {
		switch event.Type {
		case "structs.structs.EventVaultAddress", "structs.structs.EventVaultDelegationMigration", "structs.structs.EventVaultUnbondingMigration", "structs.structs.EventVaultRedelegationMigration", "structs.structs.EventInfusion", "structs.structs.EventDelete", "structs.structs.EventGrid":
		default:
			continue
		}
		// The SDK's BaseApp adds a raw mode=EndBlock attribute. It
		// belongs to the envelope, not the JSON-valued protobuf payload.
		payload := abci.Event{Type: event.Type}
		for _, attribute := range event.Attributes {
			if attribute.Key == "mode" {
				require.Equal(t, "EndBlock", attribute.Value)
				continue
			}
			payload.Attributes = append(payload.Attributes, attribute)
		}
		decoded, err := sdk.ParseTypedEvent(payload)
		require.NoError(t, err, "event: %+v", event)
		switch detail := decoded.(type) {
		case *types.EventInfusion:
			row := *detail.Infusion
			infusions[row.DestinationId+"-"+row.Address] = row
			infusionEvents++
		case *types.EventDelete:
			if brownout && detail.ObjectId == shedAllocation {
				allocationDeleted = true
			}
			if _, exists := infusions[detail.ObjectId]; exists {
				delete(infusions, detail.ObjectId)
				removedInfusions++
			}
		case *types.EventGrid:
			row := detail.GridRecord
			if row.Value == 0 {
				delete(grid, row.AttributeId)
			} else {
				grid[row.AttributeId] = row.Value
			}
			gridEvents++
		case *types.EventVaultAddress:
			associations[detail.EventVaultAddressDetail.OwnerId] = detail.EventVaultAddressDetail.VaultAddress
		case *types.EventVaultDelegationMigration:
			require.Equal(t, detail.VaultAddress, associations[detail.OwnerId], "association must precede relocation")
			oldKey := [2]string{detail.SourceAddress, detail.ValidatorAddress}
			old, exists := delegations[oldKey]
			require.True(t, exists, "source must be present exactly once")
			require.Equal(t, old.Shares.String(), detail.SharesMoved)
			newKey := [2]string{detail.VaultAddress, detail.ValidatorAddress}
			shares, err := math.LegacyNewDecFromStr(detail.DestinationShares)
			require.NoError(t, err)
			expected := old.Shares
			if previous, exists := delegations[newKey]; exists {
				expected = expected.Add(previous.Shares)
			}
			require.True(t, expected.Equal(shares), "event must conserve exact shares")
			delete(delegations, oldKey)
			delegations[newKey] = stakingtypes.NewDelegation(detail.VaultAddress, detail.ValidatorAddress, shares)
			if detail.OwnerId == playerID {
				counts["delegation"]++
			}
		case *types.EventVaultUnbondingMigration:
			row := detail.Destination
			require.Equal(t, row.DelegatorAddress, associations[detail.OwnerId])
			for _, address := range detail.SourceAddresses {
				key := [2]string{address, row.ValidatorAddress}
				_, exists := unbondings[key]
				require.True(t, exists)
				delete(unbondings, key)
			}
			unbondings[[2]string{row.DelegatorAddress, row.ValidatorAddress}] = row
			if detail.OwnerId == playerID {
				counts["unbonding"]++
				require.Len(t, detail.SourceAddresses, 2)
				require.Len(t, row.Entries, 2)
			}
		case *types.EventVaultRedelegationMigration:
			row := detail.Destination
			require.Equal(t, row.DelegatorAddress, associations[detail.OwnerId])
			for _, address := range detail.SourceAddresses {
				key := [3]string{address, row.ValidatorSrcAddress, row.ValidatorDstAddress}
				_, exists := redelegations[key]
				require.True(t, exists)
				delete(redelegations, key)
			}
			redelegations[[3]string{row.DelegatorAddress, row.ValidatorSrcAddress, row.ValidatorDstAddress}] = row
			if detail.OwnerId == playerID {
				counts["redelegation"]++
				require.Len(t, detail.SourceAddresses, 2)
				require.Len(t, row.Entries, 2)
			}
		}
	}
	require.Empty(t, counts, "retained staking records emit no relocation events")
	_, err = bApp.Commit()
	require.NoError(t, err)
	finalCtx := bApp.NewContextLegacy(true, cmtproto.Header{Height: 2, ChainID: jailGateChainID, Time: blockTime})
	after := bApp.StakingKeeper.ExportGenesis(finalCtx)
	finalInfusions := map[string]types.Infusion{}
	for _, row := range bApp.StructsKeeper.GetAllInfusion(finalCtx) {
		finalInfusions[row.DestinationId+"-"+row.Address] = row
	}
	requireInfusionEventsEqual(t, finalInfusions, infusions, "public events must reconstruct all infusion fields")
	finalGrid := map[string]uint64{}
	for _, row := range bApp.StructsKeeper.GetAllGridExport(finalCtx) {
		// Grid events intentionally use zero for both deletion and a stored zero.
		if row.Value != 0 {
			finalGrid[row.AttributeId] = row.Value
		}
	}
	require.Equal(t, finalGrid, grid, "public events must reconstruct effective grid values")
	t.Logf("infusion events=%d grid events=%d", infusionEvents, gridEvents)
	require.Zero(t, removedInfusions, "legacy backing must remain at source addresses")
	if brownout {
		require.True(t, allocationDeleted, "indexers must see the shed allocation deletion")
		_, found := bApp.StructsKeeper.GetAllocation(finalCtx, shedAllocation)
		require.False(t, found)
		cc := bApp.StructsKeeper.NewCurrentContext(finalCtx)
		player := cc.GetPlayer(playerID)
		require.Zero(t, player.GetLoad())
		require.Positive(t, player.GetCapacity(), "load shedding must not destroy surviving capacity")
	}
	if rebase {
		for _, row := range finalInfusions {
			if row.DestinationType == types.ObjectType_reactor {
				require.NoError(t, row.ValidateAssetOwner())
				reactor, found := bApp.StructsKeeper.GetReactor(finalCtx, row.DestinationId)
				require.True(t, found)
				validator, err := sdk.ValAddressFromBech32(reactor.Validator)
				require.NoError(t, err)
				backing, err := bApp.StakingKeeper.GetValidator(finalCtx, validator)
				require.NoError(t, err)
				require.True(t, backing.IsBonded() && !backing.Jailed)
				require.Equal(t, row.Fuel, row.Power)
			}
		}
	}
	require.Len(t, delegations, len(after.Delegations))
	for _, row := range after.Delegations {
		require.Equal(t, row, delegations[[2]string{row.DelegatorAddress, row.ValidatorAddress}])
	}
	require.Len(t, unbondings, len(after.UnbondingDelegations))
	for _, row := range after.UnbondingDelegations {
		require.Equal(t, row, unbondings[[2]string{row.DelegatorAddress, row.ValidatorAddress}])
	}
	require.Len(t, redelegations, len(after.Redelegations))
	for _, row := range after.Redelegations {
		require.Equal(t, row, redelegations[[3]string{row.DelegatorAddress, row.ValidatorSrcAddress, row.ValidatorDstAddress}])
	}
	requireStakingAndDistributionHealthy(t, bApp, finalCtx)
	if brownout {
		_, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 3, Time: blockTime.Add(time.Second)})
		require.NoError(t, err, "chain must continue after the upgrade brownout")
		_, err = bApp.Commit()
		require.NoError(t, err)
		continued := bApp.NewContextLegacy(true, cmtproto.Header{Height: 3, ChainID: jailGateChainID, Time: blockTime.Add(time.Second)})
		requireStakingAndDistributionHealthy(t, bApp, continued)
		_, found := bApp.StructsKeeper.GetAllocation(continued, shedAllocation)
		require.False(t, found, "the shed allocation must remain removed")
	}
}

// Test fixtures run the new binary before migration. Restore the old queue
// encoding explicitly; the production re-key is defined only for old rows.
func restoreLegacyCascadeQueueFixture(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()
	oldQueue := bApp.StructsKeeper.GetGridCascadeQueue(ctx, true)
	found := false
	for _, key := range bApp.GetStoreKeys() {
		if key.Name() != types.StoreKey {
			continue
		}
		found = true
		store := ctx.KVStore(key)
		store.Delete([]byte(types.GridCascadeQueueSequenceKey))
		for _, objectID := range oldQueue {
			store.Set(append([]byte(types.GridCascadeQueue), []byte(objectID)...), []byte{0, 0, 0, 0, 0, 0, 0, 1})
		}
	}
	require.True(t, found, "legacy queue fixture requires the structs store")
}
