package app_test

import (
	"math/rand"
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

// BaseApp recovering an index panic is not a slot-validation result. Include
// each invalid message in a real block, then prove the next valid move/build
// still uses the requested slot and leaves its neighbour intact.
func TestSignedStructPlacementRefusesInvalidSlotAndContinues(t *testing.T) {
	for _, placement := range []types.ObjectType{types.ObjectType_planet, types.ObjectType_fleet} {
		for _, action := range []string{"move", "build"} {
			t.Run(placement.String()+"/"+action, func(t *testing.T) {
				bApp, ctx := setupJailGateApp(t)
				key := secp256k1.GenPrivKey()
				address := sdk.AccAddress(key.PubKey().Address())
				cc := bApp.StructsKeeper.NewCurrentContext(ctx)
				owner := cc.UpsertPlayer(address.String())
				ownerID := owner.ID()
				fleetID := owner.GetFleet().ID()
				owner.GetFleet().GetCommandStruct()
				owner.SetPlanetId("2-99001")
				cc.CommitAll()
				// This slot test uses legacy fixed capacity. Live v0.22 backing is
				// exercised separately against real staking in struct_move_supply_test.

				bApp.StructsKeeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, ownerID), 10000000)
				kind := types.StructType{Id: 99001, Type: "Movable fixture", Category: placement, Movable: true, PossibleAmbit: types.Ambit_flag[types.Ambit_land]}
				bApp.StructsKeeper.SetStructType(ctx, kind)
				planet := types.Planet{Id: "2-99001", Owner: ownerID, Creator: address.String(), LandSlots: 2, Land: []string{"", ""}}
				fleet, found := bApp.StructsKeeper.GetFleet(ctx, fleetID)
				require.True(t, found)
				require.NotEmpty(t, fleet.CommandStruct)
				bApp.StructsKeeper.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, fleet.CommandStruct), uint64(types.StructStateBuilt|types.StructStateOnline|types.StructStateMaterialized))
				fleet.LandSlots, fleet.Land = 2, []string{"", ""}
				location := planet.Id
				if placement == types.ObjectType_fleet {
					location = fleetID
				}
				unit := types.Struct{Id: "5-99001", Index: 99001, Type: kind.Id, Owner: ownerID, Creator: address.String(),
					LocationId: location, LocationType: placement, OperatingAmbit: types.Ambit_land, Slot: 0}
				if placement == types.ObjectType_planet {
					planet.Land[0] = unit.Id
				} else {
					fleet.Land[0] = unit.Id
				}
				bApp.StructsKeeper.SetPlanet(ctx, planet)
				bApp.StructsKeeper.SetFleet(ctx, fleet)
				bApp.StructsKeeper.SetStruct(ctx, unit)
				bApp.StructsKeeper.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, unit.Id), uint64(types.StructStateBuilt|types.StructStateOnline|types.StructStateMaterialized))
				config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
				for index, slot := range []uint64{^uint64(0), 1} {
					var msg sdk.Msg
					if action == "move" {
						msg = &types.MsgStructMove{Creator: address.String(), StructId: unit.Id, LocationType: placement, Ambit: types.Ambit_land, Slot: slot}
					} else {
						msg = &types.MsgStructBuildInitiate{Creator: address.String(), PlayerId: ownerID, StructTypeId: kind.Id, OperatingAmbit: types.Ambit_land, Slot: slot}
					}
					signer := bApp.AccountKeeper.GetAccount(ctx, address)
					tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, []sdk.Msg{msg}, nil,
						ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
					require.NoError(t, err)
					wire, err := config.TxEncoder()(tx)
					require.NoError(t, err)
					block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: int64(index + 1), Time: ctx.BlockTime(), Txs: [][]byte{wire}})
					require.NoError(t, err)
					require.Len(t, block.TxResults, 1)
					result := block.TxResults[0]
					require.NotContains(t, result.Log, "panic")
					ctx = bApp.NewContext(false).WithBlockTime(ctx.BlockTime())
					current, found := bApp.StructsKeeper.GetStruct(ctx, unit.Id)
					require.True(t, found)
					if index == 0 {
						require.NotZero(t, result.Code, result.Log)
						require.Contains(t, result.Log, "doesn't have that slot available")
						require.Equal(t, unit, current)
					} else {
						require.Zero(t, result.Code, result.Log)
						var slots []string
						if placement == types.ObjectType_planet {
							p, _ := bApp.StructsKeeper.GetPlanet(ctx, planet.Id)
							slots = p.Land
						} else {
							f, _ := bApp.StructsKeeper.GetFleet(ctx, fleetID)
							slots = f.Land
						}
						if action == "move" {
							require.Equal(t, uint64(1), current.Slot)
							require.Equal(t, []string{"", unit.Id}, slots)
						} else {
							require.Equal(t, unit.Id, slots[0])
							require.NotEmpty(t, slots[1])
							require.NotEqual(t, unit.Id, slots[1])
						}
					}
					_, err = bApp.Commit()
					require.NoError(t, err)
					ctx = bApp.NewUncachedContext(false, ctx.BlockHeader()).WithBlockTime(ctx.BlockTime())
				}
			})
		}
	}
}
