package app_test

import (
	"bytes"
	"fmt"
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
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestPlanetOreConservationAcrossAppGenesisRestart(t *testing.T) {
	for _, zero := range []bool{false, true} {
		t.Run(fmt.Sprintf("zero=%v", zero), func(t *testing.T) {
			original, ctx := setupJailGateApp(t)
			remainingByID := make(map[string]uint64)
			statusByPlanet := make(map[string]types.PlanetStatus)
			// Represent unmined, partially mined and exhausted worlds, with the
			// extracted ore still owned by their players. No proof-of-work or
			// mining-handler claim is made by this restart fixture.
			for remaining := uint64(0); remaining <= types.PlanetStartingOre; remaining++ {
				address := sdk.AccAddress(bytes.Repeat([]byte{byte(0xa0 + remaining)}, 20))
				player := appendAppPlayer(t, original.StructsKeeper, ctx, address)
				cc := original.StructsKeeper.NewCurrentContext(ctx)
				planet := cc.NewPlanet(address.String(), player.Id)
				cc.GetPlayer(player.Id).SetPlanetId(planet.GetPlanetId())
				mined := uint64(types.PlanetStartingOre) - remaining
				planet.BuriedOreDecrement(mined)
				cc.GetPlayer(player.Id).StoredOreIncrement(mined)
				cc.CommitAll()
				if remaining == 0 {
					record, found := original.StructsKeeper.GetPlanet(ctx, planet.GetPlanetId())
					require.True(t, found)
					record.Status = types.PlanetStatus_complete
					original.StructsKeeper.SetPlanet(ctx, record)
				}
				statusByPlanet[planet.GetPlanetId()] = types.PlanetStatus_active
				if remaining == 0 {
					statusByPlanet[planet.GetPlanetId()] = types.PlanetStatus_complete
				}
				remainingByID[keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, planet.GetPlanetId())] = remaining
				remainingByID[keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, player.Id)] = mined
			}
			for height := int64(1); height <= 3; height++ {
				_, err := original.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: ctx.BlockTime().Add(time.Duration(height) * time.Second)})
				require.NoError(t, err)
				_, err = original.Commit()
				require.NoError(t, err)
			}
			for pass := 0; pass < 2; pass++ {
				exported, err := original.ExportAppStateAndValidators(zero, nil, nil)
				require.NoError(t, err)
				restarted, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, baseapp.SetChainID(jailGateChainID))
				require.NoError(t, err)
				restartTime := ctx.BlockTime().Add(time.Duration(10+pass*10) * time.Second)
				_, err = restarted.InitChain(&abci.RequestInitChain{AppStateBytes: exported.AppState, ChainId: jailGateChainID, InitialHeight: exported.Height, Time: restartTime, ConsensusParams: &exported.ConsensusParams})
				require.NoError(t, err)
				height := max(int64(1), exported.Height)
				restored := restarted.NewContextLegacy(false, cmtproto.Header{Height: height, ChainID: jailGateChainID, Time: restartTime})
				for id, remaining := range remainingByID {
					require.Equal(t, remaining, restarted.StructsKeeper.GetGridAttribute(restored, id), "pass %d: %s", pass, id)
				}
				for id, status := range statusByPlanet {
					planet, found := restarted.StructsKeeper.GetPlanet(restored, id)
					require.True(t, found)
					require.Equal(t, status, planet.Status)
				}
				block, err := restarted.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: restartTime})
				require.NoError(t, err)
				if zero {
					// The first-block genesis snapshot must tell indexers the same
					// balances as queries, including explicit exhausted zero rows.
					replayed := make(map[string]uint64)
					for _, event := range block.Events {
						if event.Type != "structs.structs.EventGrid" {
							continue
						}
						payload := abci.Event{Type: event.Type}
						for _, attribute := range event.Attributes {
							if attribute.Key != "mode" {
								payload.Attributes = append(payload.Attributes, attribute)
							}
						}
						decoded, err := sdk.ParseTypedEvent(payload)
						require.NoError(t, err)
						row := decoded.(*types.EventGrid).GridRecord
						if expected, found := remainingByID[row.AttributeId]; found {
							require.Equal(t, expected, row.Value)
							replayed[row.AttributeId] = row.Value
						}
					}
					require.Equal(t, remainingByID, replayed)
				}
				_, err = restarted.Commit()
				require.NoError(t, err)
				original = restarted
			}
		})
	}
}
