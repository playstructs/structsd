package app_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestAllocationPowerCleanupSurvivesGenesisAndDatabaseRestart(t *testing.T) {
	for _, zero := range []bool{false, true} {
		t.Run(fmt.Sprint(zero), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			const orphan = "5-6-999999"
			key := []byte(types.GridAttributeKey + orphan)
			bApp.StructsKeeper.SetGridAttribute(ctx, orphan, 123)
			_, err := v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, bApp.ModuleManager.GetVersionMap())
			require.NoError(t, err)
			require.Nil(t, ctx.KVStore(bApp.GetKey(types.StoreKey)).Get(key))
			_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
			require.NoError(t, err)
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
				result, err := restored.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height + step, Time: ctx.BlockTime().Add(time.Duration(step+1) * time.Second)})
				require.NoError(t, err)
				for _, event := range result.Events {
					if event.Type != "structs.structs.EventGrid" {
						continue
					}
					payload := abci.Event{Type: event.Type}
					for _, attribute := range event.Attributes {
						if attribute.Key != "mode" {
							payload.Attributes = append(payload.Attributes, attribute)
						}
					}
					parsed, err := sdk.ParseTypedEvent(payload)
					require.NoError(t, err)
					require.NotEqual(t, orphan, parsed.(*types.EventGrid).GridRecord.AttributeId, "restart cannot recreate or re-clear the deleted attribute")
				}
				_, err = restored.Commit()
				require.NoError(t, err)
				restored = newApp()
				current := restored.NewContext(true).WithBlockHeight(height + step)
				require.Nil(t, current.KVStore(restored.GetKey(types.StoreKey)).Get(key))
				for _, row := range restored.StructsKeeper.GetAllGridExport(current) {
					require.NotEqual(t, orphan, row.AttributeId)
				}
			}
		})
	}
}
