package app_test

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
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
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Export omits the slash work queue because genesis rebuilds reactor positions
// from staking. Prove that a partially processed repair queue can be completed
// by that rebuild, rather than merely disappearing with stale power intact.
func TestPendingSlashRebuiltAcrossExport(t *testing.T) {
	testPendingReactorWorkRebuiltAcrossExport(t, false)
}

func TestPendingRestorationRebuiltAcrossExport(t *testing.T) {
	testPendingReactorWorkRebuiltAcrossExport(t, true)
}

func testPendingReactorWorkRebuiltAcrossExport(t *testing.T, restoration bool) {
	for _, zeroHeight := range []bool{false, true} {
		t.Run(fmt.Sprintf("zero-height=%t", zeroHeight), func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 2)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			reactorID := jailGateReactorId(t, bApp, ctx, validator)
			otherValidator, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
			require.NoError(t, err)
			otherReactorID := jailGateReactorId(t, bApp, ctx, otherValidator)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			count := types.ReactorSlashReconcileBudget + 2
			vaults := make([]sdk.AccAddress, count)
			players := make([]string, count)
			for i := range vaults {
				address := make([]byte, 20)
				address[0] = 0x7a
				binary.BigEndian.PutUint64(address[12:], uint64(i+1))
				player := appendAppPlayer(t, bApp.StructsKeeper, ctx, sdk.AccAddress(address))
				players[i] = player.Id
				addresses, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(player.Id)
				require.NoError(t, err)
				vaults[i] = addresses[0]
				delegate(t, bApp, ctx, staking, denom, vaults[i], validator, math.NewInt(1000000))
				delegate(t, bApp, ctx, staking, denom, vaults[i], otherValidator, math.NewInt(1000000))
				portfolio, err := bApp.StructsKeeper.OwnerReactorDiversity(ctx, &types.QueryOwnerReactorDiversityRequest{OwnerId: player.Id})
				require.NoError(t, err)
				require.Positive(t, portfolio.Stored.AppliedBonus)
			}

			value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
			require.NoError(t, err)
			consensus, err := value.GetConsAddr()
			require.NoError(t, err)
			expectedFuel := uint64(950000)
			if restoration {
				expectedFuel = 1000000
				require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
				_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
				require.NoError(t, err)
				for i := 0; i < 2; i++ {
					bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
				}
				require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
				require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
				_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
				require.NoError(t, err)
			} else {
				_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
				require.NoError(t, err)
			}
			_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
			require.NoError(t, err)
			_, err = bApp.Commit()
			require.NoError(t, err)
			committed := bApp.NewContext(true).WithBlockTime(ctx.BlockTime())
			queue := bApp.StructsKeeper.GetReactorSlashReconcileQueue(committed)
			require.Len(t, queue, 1)
			require.NotEmpty(t, queue[0].Cursor)
			revision, err := bApp.StructsKeeper.GetReactorCommissionRevision(committed, reactorID)
			require.NoError(t, err)
			if !restoration {
				require.Equal(t, keeperlib.ReactorCommissionRevision{Issued: 1}, revision, "partial repair must not certify commission")
			}
			snapshot, found, err := bApp.StructsKeeper.GetReactorRepairSnapshot(committed, reactorID)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, revision.Issued, snapshot.Revision)
			require.Positive(t, snapshot.After)
			require.Greater(t, snapshot.Through, snapshot.After)
			stale, reconciled := 0, 0
			trackedVaults := map[string]bool{}
			for _, vault := range vaults {
				trackedVaults[vault.String()] = true
				row, found := bApp.StructsKeeper.GetInfusion(committed, reactorID, vault.String())
				require.True(t, found)
				if restoration {
					require.Equal(t, expectedFuel, row.Fuel)
					if row.Power == 0 {
						stale++
					} else {
						reconciled++
					}
				} else if row.Fuel == 1000000 {
					stale++
				} else if row.Fuel == expectedFuel {
					reconciled++
				} else {
					t.Fatalf("unexpected fuel before export: %d", row.Fuel)
				}
			}
			require.Positive(t, stale, "export must capture unfinished work")
			require.Positive(t, reconciled, "the first block must have processed a real batch")
			exported, err := bApp.ExportAppStateAndValidators(zeroHeight, nil, nil)
			require.NoError(t, err)
			var modules map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(exported.AppState, &modules))
			var genesis types.GenesisState
			require.NoError(t, bApp.AppCodec().UnmarshalJSON(modules[types.ModuleName], &genesis))
			exportedStale := 0
			for _, row := range genesis.InfusionList {
				if row.DestinationId != reactorID || !trackedVaults[row.Address] {
					continue
				}

				if (restoration && row.Power == 0) || (!restoration && row.Fuel == 1000000) {
					exportedStale++
				}
			}
			require.Equal(t, stale, exportedStale, "the imported bytes must contain the unreconciled positions")
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
			require.Empty(t, restarted.StructsKeeper.GetReactorSlashReconcileQueue(restored))
			rebuiltRevision, err := restarted.StructsKeeper.GetReactorCommissionRevision(restored, reactorID)
			require.NoError(t, err)
			require.Equal(t, keeperlib.ReactorCommissionRevision{}, rebuiltRevision)
			_, found, err = restarted.StructsKeeper.GetReactorRepairSnapshot(restored, reactorID)
			require.NoError(t, err)
			require.False(t, found, "genesis rebuild replaces the unfinished interval")
			for i, vault := range vaults {
				row, found := restarted.StructsKeeper.GetInfusion(restored, reactorID, vault.String())
				require.True(t, found)
				require.Equal(t, expectedFuel, row.Fuel)

				require.Equal(t, expectedFuel, row.Power)
				_, _, playerPower := row.GetPowerDistribution()
				other, found := restarted.StructsKeeper.GetInfusion(restored, otherReactorID, vault.String())
				require.True(t, found)
				require.Equal(t, uint64(1000000), other.Power)
				_, _, otherPower := other.GetPowerDistribution()
				// Independent two-reactor reduction of the configured curve:
				// bonus = floor((p+q)*p*q / (9*(p*p+q*q))). Fixture values
				// fit uint64; production calculation supports arbitrary totals.
				bonus := (playerPower + otherPower) * playerPower * otherPower / (9 * (playerPower*playerPower + otherPower*otherPower))
				portfolio, err := restarted.StructsKeeper.OwnerReactorDiversity(restored, &types.QueryOwnerReactorDiversityRequest{OwnerId: players[i]})
				require.NoError(t, err)
				require.Equal(t, bonus, portfolio.Stored.AppliedBonus)
				require.Equal(t, bonus, portfolio.UsableBonus)
				require.Equal(t, playerPower+otherPower+bonus, redelegationCapacity(t, restarted, restored, players[i]))
				owner, found := restarted.StructsKeeper.GetVaultOwner(restored, vault)
				require.True(t, found)
				require.Equal(t, players[i], owner)
				entries, more, rows := restarted.StructsKeeper.GetInfusionOwnerEntriesUpTo(restored, owner, 3)
				require.False(t, more)
				require.Equal(t, 2, rows)
				require.Len(t, entries, 2)
				require.ElementsMatch(t, []string{reactorID, otherReactorID}, []string{entries[0].DestinationId, entries[1].DestinationId})
				for _, entry := range entries {
					require.Equal(t, vault.String(), entry.Address)
				}
			}
			require.NoError(t, restarted.StructsKeeper.CheckReactorDiversityAccounting(restored))
			requireStakingAndDistributionHealthy(t, restarted, restored)
			replayKey := []byte("Reactor/commissionReplayPending")
			require.True(t, restored.KVStore(restarted.GetKey(types.StoreKey)).Has(replayKey))
			firstBlock, err := restarted.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: restartTime})
			require.NoError(t, err)
			resets, revisions := commissionEvents(t, firstBlock.Events)
			require.Equal(t, 1, resets, "both export height modes must replace the old indexed repair state")
			require.Empty(t, revisions, "rebuilt initial revisions are represented by the empty replacement")
			_, err = restarted.Commit()
			require.NoError(t, err)
			committedRestart := restarted.NewContext(true)
			require.False(t, committedRestart.KVStore(restarted.GetKey(types.StoreKey)).Has(replayKey))
			response, err := restarted.StructsKeeper.GetReactorCommissionRevision(committedRestart, reactorID)
			require.NoError(t, err)
			require.Zero(t, response.Issued)
			require.Zero(t, response.Certified)
			require.False(t, response.Issued != response.Certified)
			secondBlock, err := restarted.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height + 1, Time: restartTime.Add(time.Second)})
			require.NoError(t, err)
			resets, _ = commissionEvents(t, secondBlock.Events)
			require.Zero(t, resets, "ordinary continuation must not clear the index again")
			_, err = restarted.Commit()
			require.NoError(t, err)
		})
	}
}
