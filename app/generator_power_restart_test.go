package app_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/app"
	"structs/x/structs/types"
)

func TestGeneratorPowerQueueResumesAcrossExport(t *testing.T) {
	for _, phase := range []string{"shutdown", "restoration", "destruction"} {
		for _, zeroHeight := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/zero-height=%t", phase, zeroHeight), func(t *testing.T) {
				bApp, ctx, generatorID, _, donors := setupGeneratorQueueApp(t, types.GeneratorPowerReconcileBudget+2)
				cc := bApp.StructsKeeper.NewCurrentContext(ctx)
				if phase == "destruction" {
					cc.GetStruct(generatorID).DestroyAndCommit()
				} else {
					cc.GetStruct(generatorID).GoOffline()
				}
				cc.CommitAll()
				if phase == "restoration" {
					bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
					bApp.StructsKeeper.ProcessGeneratorPowerQueue(ctx)
					cc = bApp.StructsKeeper.NewCurrentContext(ctx)
					cc.GetStruct(generatorID).GoOnline()
					cc.CommitAll()
				}
				_, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
				require.NoError(t, err)
				_, err = bApp.Commit()
				require.NoError(t, err)
				committed := bApp.NewContext(true)
				snapshot, queued, err := bApp.StructsKeeper.GetGeneratorPowerSnapshot(committed, generatorID)
				require.NoError(t, err)
				require.True(t, queued)
				require.Positive(t, snapshot.After)
				require.Greater(t, snapshot.Through, snapshot.After)
				expected := map[string]types.Infusion{}
				for _, row := range bApp.StructsKeeper.GetAllInfusion(committed) {
					if row.DestinationId == generatorID {
						expected[row.Address] = row
					}
				}
				supply := bApp.BankKeeper.GetSupply(committed, "ualpha").Amount
				exported, err := bApp.ExportAppStateAndValidators(zeroHeight, nil, nil)
				require.NoError(t, err)
				var modules map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(exported.AppState, &modules))
				var genesis types.GenesisState
				require.NoError(t, bApp.AppCodec().UnmarshalJSON(modules[types.ModuleName], &genesis))
				require.Contains(t, genesis.GeneratorPowerQueue, generatorID)
				restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
				require.NoError(t, err)
				restartTime := ctx.BlockTime().Add(time.Second)
				_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: restartTime, ConsensusParams: &exported.ConsensusParams})
				require.NoError(t, err)
				height := exported.Height
				if height == 0 {
					height = 1
				}
				restored := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: restartTime})
				for _, donor := range donors {
					vault, err := types.VaultAddress(donor.Id)
					require.NoError(t, err)
					row, found := restarted.StructsKeeper.GetInfusion(restored, generatorID, vault.String())
					old, existed := expected[vault.String()]
					require.Equal(t, existed, found)
					if existed {
						require.Equal(t, old, row, "genesis must retain both finished and unfinished positions")
					}
				}
				// Import rebuilds ordinals and starts a fresh finite pass. Already
				// processed flags make replay idempotent, including zero-power fuel.
				updates, deletes := map[string]int{}, map[string]int{}
				replayed := map[string]types.Infusion{}
				for pass := int64(0); pass < 2; pass++ {
					block, err := restarted.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height + pass, Time: restartTime.Add(time.Duration(pass) * time.Second)})
					require.NoError(t, err)
					for _, event := range block.Events {
						if event.Type != "structs.structs.EventInfusion" && event.Type != "structs.structs.EventDelete" {
							continue
						}
						payload := abci.Event{Type: event.Type}
						mode := ""
						for _, attribute := range event.Attributes {
							if attribute.Key != "mode" {
								payload.Attributes = append(payload.Attributes, attribute)
							} else {
								mode = attribute.Value
							}
						}
						decoded, err := sdk.ParseTypedEvent(payload)
						require.NoError(t, err)
						switch detail := decoded.(type) {
						case *types.EventInfusion:
							row := detail.Infusion
							if row.DestinationId == generatorID {
								if mode == "BeginBlock" {
									requireInfusionEventEqual(t, expected[row.Address], *row, "genesis replay describes the imported partial state")
									replayed[row.Address] = *row
									continue
								}
								require.Equal(t, "EndBlock", mode)
								updates[row.Address]++
								require.Equal(t, uint64(1000), row.Fuel)
								require.Equal(t, uint64(2), row.Ratio)
								require.Equal(t, phase == "shutdown", row.GeneratorDisabled)
							}
						case *types.EventDelete:
							if strings.HasPrefix(detail.ObjectId, generatorID+"-") {
								deletes[strings.TrimPrefix(detail.ObjectId, generatorID+"-")]++
							}
						}
					}
					_, err = restarted.Commit()
					require.NoError(t, err)
				}
				finished := restarted.NewContext(true)
				_, queued, err = restarted.StructsKeeper.GetGeneratorPowerSnapshot(finished, generatorID)
				require.NoError(t, err)
				require.False(t, queued)
				if zeroHeight {
					requireInfusionEventsEqual(t, expected, replayed, "complete replay matches stored infusions")
				} else {
					require.Empty(t, replayed)
				}
				if phase == "destruction" {
					require.Len(t, deletes, 2)
					require.Empty(t, updates)
				} else {
					require.Len(t, updates, 2, "only the two unfinished positions change after import")
					require.Empty(t, deletes)
				}
				for _, count := range updates {
					require.Equal(t, 1, count)
				}
				for _, count := range deletes {
					require.Equal(t, 1, count)
				}
				for _, donor := range donors {
					vault, err := types.VaultAddress(donor.Id)
					require.NoError(t, err)
					row, found := restarted.StructsKeeper.GetInfusion(finished, generatorID, vault.String())
					power := uint64(0)
					if phase == "destruction" {
						require.False(t, found)
					} else {
						require.True(t, found)
						require.Equal(t, uint64(1000), row.Fuel)
						require.Equal(t, uint64(2), row.Ratio)
						require.Equal(t, phase == "shutdown", row.GeneratorDisabled)
						if phase == "restoration" {
							power = 2000
						}
						require.Equal(t, power, row.Power)
					}
					require.Equal(t, power, restarted.StructsKeeper.NewCurrentContext(finished).GetPlayer(donor.Id).GetCapacity())
					require.True(t, restarted.BankKeeper.GetBalance(finished, vault, "ualpha").IsZero())
				}
				require.Equal(t, supply, restarted.BankKeeper.GetSupply(finished, "ualpha").Amount)
			})
		}
	}
}
