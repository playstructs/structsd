package app_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// A future stored anchor must not turn a cheap proof into a successful raid.
// This seeds a clock inconsistency deliberately; normal genesis import sends
// fleets home and clears raid clocks, so it does not itself create this state.
func TestSignedRaidClockRefusesFutureAndContinuesAtRealAge(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	key := secp256k1.GenPrivKey()
	address := sdk.AccAddress(key.PubKey().Address())
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	attacker := cc.UpsertPlayer(address.String())
	victim := cc.UpsertPlayer(sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String())
	attackerID, victimID := attacker.ID(), victim.ID()
	home := cc.NewPlanet(address.String(), attackerID)
	target := cc.NewPlanet(victim.GetPrimaryAddress(), victimID)
	attacker.SetPlanetId(home.ID())
	victim.SetPlanetId(target.ID())
	fleetID := attacker.GetFleet().GetFleet().Id
	cc.CommitAll()
	// Isolate the clock from power admission. The native supply tests cover
	// live reactor backing independently; this fixture supplies fixed power.

	bApp.StructsKeeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attackerID), 10000000)
	fleet, found := bApp.StructsKeeper.GetFleet(ctx, fleetID)
	require.True(t, found)
	fleet.LocationId, fleet.LocationType, fleet.Status = target.ID(), types.ObjectType_planet, types.FleetStatus_away
	bApp.StructsKeeper.SetFleet(ctx, fleet)
	planet, found := bApp.StructsKeeper.GetPlanet(ctx, target.ID())
	require.True(t, found)
	planet.LocationListStart, planet.LocationListLast, planet.LocationListCount = fleetID, fleetID, 1
	bApp.StructsKeeper.SetPlanet(ctx, planet)
	clock := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, target.ID())
	victimOre := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, victimID)
	attackerOre := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, attackerID)
	bApp.StructsKeeper.SetPlanetAttribute(ctx, clock, 2)
	bApp.StructsKeeper.SetGridAttribute(ctx, victimOre, 50)
	var nonce, proof string
	for n := 0; ; n++ {
		nonce = fmt.Sprint(n)
		proof = types.HashBuild(fmt.Sprintf("%s@%sRAID2NONCE%s", fleetID, target.ID(), nonce))
		if strings.HasPrefix(proof, "0") && !strings.HasPrefix(proof, "00") {
			break
		}
	}
	config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
	for height := int64(1); height <= 27; height++ {
		var txs [][]byte
		if height == 1 || height == 2 || height == 27 {
			msg := &types.MsgPlanetRaidComplete{Creator: address.String(), FleetId: fleetID, Nonce: nonce, Proof: proof}
			signer := bApp.AccountKeeper.GetAccount(ctx, address)
			tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, []sdk.Msg{msg}, nil,
				ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, err)
			wire, err := config.TxEncoder()(tx)
			require.NoError(t, err)
			txs = [][]byte{wire}
		}
		block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: ctx.BlockTime(), Txs: txs})
		require.NoError(t, err)
		ctx = bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
		if len(txs) != 0 {
			require.Len(t, block.TxResults, 1)
			result := block.TxResults[0]
			require.NotContains(t, result.Log, "panic")
			if height < 27 {
				require.NotZero(t, result.Code, result.Log)
				if height == 1 {
					require.Contains(t, result.Log, "precedes start block")
				} else {
					require.Contains(t, result.Log, "work")
				}
				for _, event := range result.Events {
					require.NotContains(t, []string{"structs.structs.EventOreTheft", "structs.structs.EventHashSuccess", "structs.structs.EventRaid"}, event.Type)
				}
			} else {
				require.Zero(t, result.Code, result.Log)
				thefts, raids, hashes := 0, 0, 0
				for _, event := range result.Events {
					switch event.Type {
					case "structs.structs.EventOreTheft":
						thefts++
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						detail := decoded.(*types.EventOreTheft).EventOreTheftDetail
						require.Equal(t, victimID, detail.VictimPlayerId)
						require.Equal(t, victim.GetPrimaryAddress(), detail.VictimPrimaryAddress)
						require.Equal(t, attackerID, detail.ThiefPlayerId)
						require.Equal(t, address.String(), detail.ThiefPrimaryAddress)
						require.Equal(t, uint64(50), detail.Amount)
					case "structs.structs.EventRaid":
						raids++
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						detail := decoded.(*types.EventRaid).EventRaidDetail
						require.Equal(t, fleetID, detail.FleetId)
						require.Equal(t, target.ID(), detail.PlanetId)
						require.Equal(t, types.RaidStatus_raidSuccessful, detail.Status)
						require.Equal(t, uint64(50), detail.SeizedOre)
					case "structs.structs.EventHashSuccess":
						hashes++
						decoded, err := sdk.ParseTypedEvent(event)
						require.NoError(t, err)
						detail := decoded.(*types.EventHashSuccess).EventHashSuccessDetail
						require.Equal(t, address.String(), detail.CallerAddress)
						require.Equal(t, "raid", detail.Category)
						require.Equal(t, fleetID, detail.ObjectId)
						require.Equal(t, target.ID(), detail.PlanetId, "proof event must retain the raided planet after fleet recall")
						require.Equal(t, uint64(1), detail.Difficulty)
					}
				}
				require.Equal(t, 1, thefts)
				require.Equal(t, 1, raids)
				require.Equal(t, 1, hashes)
			}
		}
		if height < 27 {
			require.Equal(t, uint64(50), bApp.StructsKeeper.GetGridAttribute(ctx, victimOre))
			require.Zero(t, bApp.StructsKeeper.GetGridAttribute(ctx, attackerOre))
			require.Equal(t, uint64(2), bApp.StructsKeeper.GetPlanetAttribute(ctx, clock))
			current, found := bApp.StructsKeeper.GetFleet(ctx, fleetID)
			require.True(t, found)
			require.Equal(t, fleet, current)
		} else {
			require.Zero(t, bApp.StructsKeeper.GetGridAttribute(ctx, victimOre))
			require.Equal(t, uint64(50), bApp.StructsKeeper.GetGridAttribute(ctx, attackerOre))
			current, found := bApp.StructsKeeper.GetFleet(ctx, fleetID)
			require.True(t, found)
			require.Equal(t, home.ID(), current.LocationId)
			require.Equal(t, types.FleetStatus_onStation, current.Status)
		}
		_, err = bApp.Commit()
		require.NoError(t, err)
		ctx = bApp.NewUncachedContext(false, ctx.BlockHeader()).WithBlockTime(ctx.BlockTime())
	}
}
