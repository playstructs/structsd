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

func seedLegacyAllocationType(t *testing.T, bApp *app.App, ctx sdk.Context) types.Allocation {
	t.Helper()
	k := bApp.StructsKeeper
	owner := appendAppPlayer(t, k, ctx, sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()))
	k.SetSubstation(ctx, types.Substation{Id: "4-970", Owner: owner.Id})
	k.SetSubstationCount(ctx, 971)
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.Id), 1000)
	cc := k.NewCurrentContext(ctx)
	allocation, err := cc.NewAllocation(types.AllocationType_static, owner.Id, "4-970", owner.Creator, owner.Id, 300)
	require.NoError(t, err)
	row := allocation.GetAllocation()
	cc.CommitAll()
	row.Type = 500
	_, err = k.SetAllocationOnly(ctx, row)
	require.NoError(t, err)
	return row
}

func TestLegacyAllocationTypePhasePreservesAllOtherStores(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	row := seedLegacyAllocationType(t, bApp, ctx)
	before := applicationAssetStateRows(bApp, ctx)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, v22.MigrateLegacyAllocationTypes(ctx, delegationUpgradeKeepers(bApp)))
	after := applicationAssetStateRows(bApp, ctx)
	key := types.StoreKey + "/" + types.AllocationKey + row.Id
	delete(before, key)
	delete(after, key)
	require.Equal(t, before, after, "only the type field changes; bank, auth, all indices and permissions remain exact")
	row.Type = types.AllocationType_static
	actual, found := bApp.StructsKeeper.GetAllocation(ctx, row.Id)
	require.True(t, found)
	require.Equal(t, row, actual)
	require.Len(t, ctx.EventManager().Events(), 1)
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	allocation, found := cc.GetAllocation(row.Id)
	require.True(t, found)
	_, err := allocation.SetDynamicPower(301)
	require.Error(t, err, "repair must not grant resize behavior")
}

func TestLegacyAllocationTypeRepairRollsBackWithLaterUpgradeFailure(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	row := seedLegacyAllocationType(t, bApp, ctx)
	versions := bApp.ModuleManager.GetVersionMap()
	versions[banktypes.ModuleName]--
	config := module.NewConfigurator(bApp.AppCodec(), nil, nil)
	failure := errors.New("allocation type repair late migration failure")
	called := false
	require.NoError(t, config.RegisterMigration(banktypes.ModuleName, versions[banktypes.ModuleName], func(migrationCtx sdk.Context) error {
		called = true
		actual, found := bApp.StructsKeeper.GetAllocation(migrationCtx, row.Id)
		require.True(t, found)
		require.Equal(t, types.AllocationType_static, actual.Type)
		return failure
	}))
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	before := applicationAssetStateRows(bApp, ctx)
	_, err := v22.CreateUpgradeHandler(bApp.ModuleManager, config, delegationUpgradeKeepers(bApp))(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
	require.ErrorIs(t, err, failure)
	require.True(t, called)
	require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
	require.Empty(t, ctx.EventManager().Events())
}

func TestLegacyAllocationTypeUpgradeEventsBaselineAndGenesisRestart(t *testing.T) {
	for _, zero := range []bool{false, true} {
		t.Run(fmt.Sprint(zero), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			row := seedLegacyAllocationType(t, bApp, ctx)
			const name = "test-legacy-allocation-type"
			require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: name, Height: 2}))
			_, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
			require.NoError(t, err)
			_, err = bApp.Commit()
			require.NoError(t, err)
			bApp.UpgradeKeeper.SetUpgradeHandler(name, func(goCtx context.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
				return v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(goCtx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
			})
			block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: ctx.BlockTime().Add(time.Second)})
			require.NoError(t, err)
			count := 0
			row.Type = types.AllocationType_static
			for _, event := range block.Events {
				if event.Type != "structs.structs.EventAllocation" {
					continue
				}
				parsed, err := sdk.ParseTypedEvent(event)
				require.NoError(t, err)
				actual := parsed.(*types.EventAllocation).Allocation
				if actual.Id == row.Id {
					require.Equal(t, row, *actual)
					count++
				}
			}
			require.Equal(t, 1, count, "one replacement event reaches the scheduled upgrade block")
			_, err = bApp.Commit()
			require.NoError(t, err)
			current := bApp.NewContext(true).WithBlockHeight(2)
			repaired, found := bApp.StructsKeeper.GetAllocation(current, row.Id)
			require.True(t, found)
			require.Equal(t, row, repaired)
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
				actual, found := restored.StructsKeeper.GetAllocation(restored.NewContext(true).WithBlockHeight(height+step), row.Id)
				require.True(t, found)
				require.Equal(t, row, actual)
			}
		})
	}
}
