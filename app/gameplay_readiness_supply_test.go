package app_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// The same valid action must work with healthy or independently backed power,
// refuse before mutation during sole-source jail, and recover without waiting
// for the deferred grid repair. Recovery actions deliberately remain available.
func TestGameplayHandlersRespectLiveSupplyAndPermitRecovery(t *testing.T) {
	for _, action := range []string{"activate", "stealth-on", "stealth-off", "defense-set", "defense-clear", "build", "build-complete", "mine", "refine", "fleet-move", "explore", "deactivate", "deactivate-batch", "cancel", "trash", "infuse"} {
		for _, mode := range []string{"healthy", "jailed", "backup", "recovered"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				bApp, ctx := setupJailGateAppWithValidators(t, 2)
				ctx = ctx.WithBlockHeight(100)
				address, _, playerID := newTransferPlayer(t, bApp, ctx)
				k := bApp.StructsKeeper
				server := keeperlib.NewMsgServerImpl(k)
				validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
				require.NoError(t, err)
				denom, err := bApp.StakingKeeper.BondDenom(ctx)
				require.NoError(t, err)
				for i, validator := range validators {
					if i != 0 && mode != "backup" {
						continue
					}
					operator, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
					require.NoError(t, err)
					delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, address, operator, math.NewInt(10000000))
				}
				cc := k.NewCurrentContext(ctx)
				owner := cc.GetPlayer(playerID)
				require.NoError(t, owner.AttemptPlanetExplore())
				owner.GetFleet().MigrateToNewPlanet(owner.GetPlanet())
				kindID, ambit := uint64(6), types.Ambit_air // shipped Stealth Bomber, also able to defend
				switch action {
				case "mine":
					kindID, ambit = 14, types.Ambit_land
				case "refine":
					kindID, ambit = 15, types.Ambit_land
				case "infuse":
					kindID, ambit = 20, types.Ambit_land
				}
				kind, found := cc.GetStructType(kindID)
				require.True(t, found)
				unit, err := cc.InitiateStruct(address.String(), owner, kind, ambit, 0)
				require.NoError(t, err)
				unitID := unit.ID()
				if action != "cancel" && action != "build-complete" {
					owner.StructsLoadDecrement(kind.GetStructType().BuildDraw)
					unit.StatusAddBuilt()
					unit.GoOnline()
				}
				if action == "activate" {
					unit.GoOffline()
				}
				if action == "stealth-off" {
					unit.StatusAddHidden()
				}
				fleetID, planetID := owner.GetFleetId(), owner.GetPlanetId()
				command := owner.GetFleet().GetCommandStruct().GetStruct()
				if action == "explore" {
					cc.SetGridAttribute(owner.GetPlanet().BuriedOreAttributeId, 0)
				}
				if action == "refine" {
					owner.StoredOreIncrement(1)
				}
				vaults, err := cc.EnsureVaults(playerID)
				require.NoError(t, err)
				cc.CommitAll()
				if action == "infuse" {
					fundRedelegationAccount(t, bApp, ctx, vaults[0], sdk.NewCoins(sdk.NewInt64Coin("ualpha", 2000000)))
				}
				if action == "defense-clear" {
					k.SetStructDefender(ctx, command.Id, command.Index, unitID)
				}
				destination := ""
				if action == "fleet-move" {
					_, _, other := newTransferPlayer(t, bApp, ctx)
					otherCC := k.NewCurrentContext(ctx)
					otherOwner := otherCC.GetPlayer(other)
					require.NoError(t, otherOwner.AttemptPlanetExplore())
					otherOwner.GetFleet().MigrateToNewPlanet(otherOwner.GetPlanet())
					destination = otherOwner.GetPlanetId()
					otherCC.CommitAll()
				}

				ctx = ctx.WithBlockHeight(1000000).WithEventManager(sdk.NewEventManager())
				owner = k.NewCurrentContext(ctx).GetPlayer(playerID)
				rawCapacity := owner.GetCapacity()
				require.True(t, owner.IsOnline())
				if mode != "healthy" {
					consensus, err := validators[0].GetConsAddr()
					require.NoError(t, err)
					require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
					if mode == "recovered" {
						require.True(t, owner.IsOffline())
						require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
					}
				}
				require.Equal(t, rawCapacity, owner.GetCapacity())
				require.Equal(t, mode != "jailed", owner.IsOnline())
				before := applicationAssetStateRows(bApp, ctx)
				infusions := k.GetAllInfusion(ctx)
				vaultBefore := bApp.BankKeeper.GetBalance(ctx, vaults[0], "ualpha").Amount
				supplyBefore := bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount
				proof, nonce := "", ""
				if action == "build-complete" || action == "mine" || action == "refine" {
					category, start, difficulty := "BUILD", unit.GetBlockStartBuild(), kind.GetStructType().BuildDifficulty
					if action == "mine" {
						category, start, difficulty = "MINE", unit.GetPlanet().GetBlockStartOreMine(), kind.GetStructType().OreMiningDifficulty
					}
					if action == "refine" {
						category, start, difficulty = "REFINE", unit.GetPlanet().GetBlockStartOreRefine(), kind.GetStructType().OreRefiningDifficulty
					}
					age := uint64(ctx.BlockHeight()) - start
					require.Equal(t, 1, types.CalculateDifficulty(age, difficulty), "age the work; do not alter production difficulty")
					for i := 0; i < 10000; i++ {
						nonce = fmt.Sprint(i)
						input := fmt.Sprintf("%s%s%dNONCE%s", unitID, category, start, nonce)
						proof = types.HashBuild(input)
						if valid, _ := types.HashBuildAndCheckDifficulty(input, proof, age, difficulty); valid {
							break
						}
						proof = ""
					}
					require.NotEmpty(t, proof)
				}
				switch action {
				case "activate":
					_, err = server.StructActivate(ctx, &types.MsgStructActivate{Creator: address.String(), StructId: unitID})
				case "stealth-on":
					_, err = server.StructStealthActivate(ctx, &types.MsgStructStealthActivate{Creator: address.String(), StructId: unitID})
				case "stealth-off":
					_, err = server.StructStealthDeactivate(ctx, &types.MsgStructStealthDeactivate{Creator: address.String(), StructId: unitID})
				case "defense-set":
					_, err = server.StructDefenseSet(ctx, &types.MsgStructDefenseSet{Creator: address.String(), DefenderStructId: unitID, ProtectedStructId: command.Id})
				case "defense-clear":
					_, err = server.StructDefenseClear(ctx, &types.MsgStructDefenseClear{Creator: address.String(), DefenderStructId: unitID})
				case "build":
					_, err = server.StructBuildInitiate(ctx, &types.MsgStructBuildInitiate{Creator: address.String(), PlayerId: playerID, StructTypeId: 6, OperatingAmbit: types.Ambit_air, Slot: 1})
				case "build-complete":
					_, err = server.StructBuildComplete(ctx, &types.MsgStructBuildComplete{Creator: address.String(), StructId: unitID, Proof: proof, Nonce: nonce})
				case "mine":
					_, err = server.StructOreMinerComplete(ctx, &types.MsgStructOreMinerComplete{Creator: address.String(), StructId: unitID, Proof: proof, Nonce: nonce})
				case "refine":
					_, err = server.StructOreRefineryComplete(ctx, &types.MsgStructOreRefineryComplete{Creator: address.String(), StructId: unitID, Proof: proof, Nonce: nonce})
				case "fleet-move":
					_, err = server.FleetMove(ctx, &types.MsgFleetMove{Creator: address.String(), FleetId: fleetID, DestinationLocationId: destination})
				case "explore":
					_, err = server.PlanetExplore(ctx, &types.MsgPlanetExplore{Creator: address.String(), PlayerId: playerID})
				case "deactivate":
					_, err = server.StructDeactivate(ctx, &types.MsgStructDeactivate{Creator: address.String(), StructId: unitID})
				case "deactivate-batch":
					_, err = server.StructDeactivateBatch(ctx, &types.MsgStructDeactivateBatch{Creator: address.String(), StructId: []string{unitID}})
				case "cancel":
					_, err = server.StructBuildCancel(ctx, &types.MsgStructBuildCancel{Creator: address.String(), StructId: unitID})
				case "trash":
					_, err = server.StructTrash(ctx, &types.MsgStructTrash{Creator: address.String(), StructId: unitID})
				case "infuse":
					_, err = server.StructGeneratorInfuse(ctx, &types.MsgStructGeneratorInfuse{Creator: address.String(), StructId: unitID, InfuseAmount: "2000000ualpha"})
				}
				recovery := action == "deactivate" || action == "deactivate-batch" || action == "cancel" || action == "trash" || action == "infuse"
				if mode == "jailed" && !recovery {
					var powerError *types.PlayerPowerError
					require.ErrorAs(t, err, &powerError)
					require.Equal(t, "offline", powerError.Reason)
					require.Equal(t, before, applicationAssetStateRows(bApp, ctx), "power refusal precedes canonical mutation")
					for _, event := range ctx.EventManager().Events() {
						require.Equal(t, "structs.structs.EventAddressActivity", event.Type, "refusal may announce caller activity only")
					}
				} else {
					require.NoError(t, err)
					after := k.NewCurrentContext(ctx)
					switch action {
					case "activate":
						require.True(t, after.GetStruct(unitID).IsOnline())
					case "stealth-on", "stealth-off":
						require.Equal(t, action == "stealth-on", after.GetStruct(unitID).IsHidden())
					case "defense-set", "defense-clear":
						index := k.GetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, unitID))
						if action == "defense-set" {
							require.Equal(t, command.Index, index)
						} else {
							require.Zero(t, index)
						}
					case "build":
						require.NotEmpty(t, after.GetPlayer(playerID).GetFleet().GetFleet().Air[1])
					case "build-complete":
						require.True(t, after.GetStruct(unitID).IsBuilt())
						require.True(t, after.GetStruct(unitID).IsOnline())
					case "mine":
						require.Equal(t, uint64(1), after.GetPlayer(playerID).GetStoredOre())
						require.Equal(t, uint64(types.PlanetStartingOre-1), after.GetPlayer(playerID).GetPlanet().GetBuriedOre())
					case "refine":
						require.Zero(t, after.GetPlayer(playerID).GetStoredOre())
						require.Equal(t, vaultBefore.AddRaw(1000000).String(), bApp.BankKeeper.GetBalance(ctx, vaults[0], "ualpha").Amount.String())
					case "infuse":
						require.True(t, after.GetPlayer(playerID).IsOnline(), "generator funding must restore an offline owner's usable power")
						require.Equal(t, vaultBefore.SubRaw(2000000).String(), bApp.BankKeeper.GetBalance(ctx, vaults[0], "ualpha").Amount.String())
						require.Equal(t, supplyBefore.SubRaw(2000000).String(), bApp.BankKeeper.GetSupply(ctx, "ualpha").Amount.String())
						require.Equal(t, uint64(2000000), after.GetInfusion(unitID, vaults[0].String()).GetFuel())
					case "fleet-move":
						require.Equal(t, destination, after.GetPlayer(playerID).GetFleet().GetLocationId())
					case "explore":
						require.NotEqual(t, planetID, after.GetPlayer(playerID).GetPlanetId())
					case "deactivate", "deactivate-batch":
						require.True(t, after.GetStruct(unitID).IsOffline())
					case "cancel", "trash":
						require.True(t, after.GetStruct(unitID).IsDestroyed())
					}
				}
				if action != "infuse" {
					require.Equal(t, infusions, k.GetAllInfusion(ctx), "gameplay readiness does not settle deferred reactor accounting")
				} else {
					for _, before := range infusions {
						after, found := k.GetInfusion(ctx, before.DestinationId, before.Address)
						require.True(t, found)
						require.Equal(t, before, after, "funding a generator must not rewrite deferred reactor infusions")
					}
				}
			})
		}
	}
}
