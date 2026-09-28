package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func seedStaleSourceShare(t *testing.T, bApp *app.App, ctx sdk.Context) (string, []string) {
	t.Helper()
	k := bApp.StructsKeeper
	newPlayer := func() types.Player {
		return appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
	}
	owner, first, second := newPlayer(), newPlayer(), newPlayer()
	const source, destination = "4-970", "4-971"
	k.SetSubstation(ctx, types.Substation{Id: source, Owner: owner.Id})
	k.SetSubstation(ctx, types.Substation{Id: destination, Owner: owner.Id})
	k.SetSubstationCount(ctx, 972)
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.Id), 1000)
	cc := k.NewCurrentContext(ctx)
	_, err := cc.NewAllocation(types.AllocationType_dynamic, owner.Id, source, owner.Creator, owner.Id, 1000)
	require.NoError(t, err)
	cc.GetPlayer(first.Id).MigrateSubstation(source)
	cc.GetPlayer(second.Id).MigrateSubstation(source)
	_, err = cc.NewAllocation(types.AllocationType_dynamic, source, destination, owner.Creator, owner.Id, 300)
	require.NoError(t, err)
	cc.CommitAll()
	attribute := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, source)
	k.SetGridAttribute(ctx, attribute, 500) // pre-fix: outgoing 300 was not deducted
	return attribute, []string{first.Id, second.Id}
}

func TestSubstationShareRepairRollsBackWithLaterUpgradeFailure(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	attribute, _ := seedStaleSourceShare(t, bApp, ctx)
	versions := bApp.ModuleManager.GetVersionMap()
	versions[banktypes.ModuleName]--
	config := module.NewConfigurator(bApp.AppCodec(), nil, nil)
	failure := errors.New("share repair late migration failure")
	called := false
	require.NoError(t, config.RegisterMigration(banktypes.ModuleName, versions[banktypes.ModuleName], func(migrationCtx sdk.Context) error {
		called = true
		require.Equal(t, uint64(300), bApp.StructsKeeper.GetGridAttribute(migrationCtx, attribute))
		return failure
	}))
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	before := applicationAssetStateRows(bApp, ctx)
	_, err := v22.CreateUpgradeHandler(bApp.ModuleManager, config, delegationUpgradeKeepers(bApp))(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
	require.ErrorIs(t, err, failure)
	require.True(t, called)
	require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
	require.Empty(t, ctx.EventManager().Events())
	_, err = v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, bApp.ModuleManager.GetVersionMap())
	require.NoError(t, err)
	require.Equal(t, uint64(300), bApp.StructsKeeper.GetGridAttribute(ctx, attribute))
}

func TestSubstationShareRepairUpgradeEventsAndGenesisRestart(t *testing.T) {
	for _, zero := range []bool{false, true} {
		t.Run(fmt.Sprint(zero), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			attribute, players := seedStaleSourceShare(t, bApp, ctx)
			const name = "test-substation-share-repair"
			require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: name, Height: 2}))
			_, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
			require.NoError(t, err)
			_, err = bApp.Commit()
			require.NoError(t, err)
			bApp.UpgradeKeeper.SetUpgradeHandler(name, func(goCtx context.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
				require.Equal(t, uint64(500), bApp.StructsKeeper.GetGridAttribute(goCtx, attribute))
				return v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(goCtx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
			})
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: ctx.BlockTime().Add(time.Second)})
			require.NoError(t, err)
			count := 0
			for _, event := range block.Events {
				if event.Type != "structs.structs.EventGrid" {
					continue
				}
				parsed, err := sdk.ParseTypedEvent(event)
				require.NoError(t, err)
				row := parsed.(*types.EventGrid).GridRecord
				if row.AttributeId == attribute {
					require.Equal(t, uint64(300), row.Value)
					count++
				}
			}
			require.Equal(t, 1, count, "upgrade repair reaches ABCI exactly once")
			_, err = bApp.Commit()
			require.NoError(t, err)
			exported, err := bApp.ExportAppStateAndValidators(zero, nil, nil)
			require.NoError(t, err)
			db := dbm.NewMemDB()
			newApp := func() *app.App {
				restored, err := app.New(log.NewNopLogger(), db, nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
				require.NoError(t, err)
				return restored
			}
			restored := newApp()
			_, err = restored.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: ctx.BlockTime(), ConsensusParams: &exported.ConsensusParams})
			require.NoError(t, err)
			height := max(int64(1), exported.Height)
			for step := int64(0); step < 2; step++ {
				_, err = restored.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height + step, Time: ctx.BlockTime().Add(time.Duration(step+2) * time.Second)})
				require.NoError(t, err)
				_, err = restored.Commit()
				require.NoError(t, err)
				restored = newApp()
				current := restored.NewContext(true).WithBlockHeight(height + step)
				require.Equal(t, uint64(300), restored.StructsKeeper.GetGridAttribute(current, attribute))
				cc := restored.StructsKeeper.NewCurrentContext(current)
				for _, id := range players {
					require.Equal(t, uint64(300), cc.GetPlayer(id).GetCapacitySecondary())
				}
			}
		})
	}
}

func TestSubstationLossUpgradeDefersTeardownToNormalEndBlock(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	attribute, _ := seedStaleSourceShare(t, bApp, ctx)
	const source = "4-970"
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	outgoing := cc.GetAllAllocationBySource(source)
	require.Len(t, outgoing, 1)
	id := outgoing[0].ID()
	// Seed a valid pre-loss commitment: 950 exported against 1000 incoming.
	_, err := outgoing[0].SetPower(950)
	require.NoError(t, err)
	cc.CommitAll()
	bApp.StructsKeeper.SetGridAttribute(ctx, attribute, 25)
	const name = "test-substation-operating-loss"
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: name, Height: 2}))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	checked := false
	bApp.UpgradeKeeper.SetUpgradeHandler(name, func(goCtx context.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
		result, err := v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(goCtx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
		require.NoError(t, err)
		require.Contains(t, bApp.StructsKeeper.GetGridCascadeQueueExport(goCtx), source)
		_, found := bApp.StructsKeeper.GetAllocation(goCtx, id)
		require.True(t, found, "migration only queues; it does not tear down")
		require.Zero(t, bApp.StructsKeeper.GetGridAttribute(goCtx, attribute))
		checked = true
		return result, nil
	})
	block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: ctx.BlockTime().Add(time.Second)})
	require.NoError(t, err)
	require.True(t, checked)
	_, err = bApp.Commit()
	require.NoError(t, err)
	current := bApp.NewContext(true).WithBlockHeight(2)
	_, found := bApp.StructsKeeper.GetAllocation(current, id)
	require.False(t, found, "ordinary EndBlock cascade sheds the allocation")
	require.Equal(t, uint64(450), bApp.StructsKeeper.GetGridAttribute(current, attribute))
	deletions := 0
	for _, event := range block.Events {
		if event.Type != "structs.structs.EventDelete" {
			continue
		}
		// BaseApp's raw mode label is an envelope field, not protobuf JSON.
		payload := abci.Event{Type: event.Type}
		mode := ""
		for _, attribute := range event.Attributes {
			if attribute.Key == "mode" {
				mode = attribute.Value
			} else {
				payload.Attributes = append(payload.Attributes, attribute)
			}
		}
		parsed, err := sdk.ParseTypedEvent(payload)
		require.NoError(t, err)
		if parsed.(*types.EventDelete).ObjectId == id {
			require.Equal(t, "EndBlock", mode)
			deletions++
		}
	}
	require.Equal(t, 1, deletions, "normal deletion event reaches indexers once")
}
