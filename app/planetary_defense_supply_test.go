package app_test

import (
	storetypes "cosmossdk.io/store/types"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Stored planetary counters belong to deferred teardown accounting. They must
// not grant combat effects after the same live power loss that stops counters
// by individual defending structs.
func TestPlanetaryDefenseRequiresLiveOwnerSupply(t *testing.T) {
	for _, defense := range []string{"cannon", "jamming"} {
		for _, mode := range []string{"healthy", "jailed", "backup", "recovered", "crowded", "reactor-crowded"} {
			t.Run(defense+"/"+mode, func(t *testing.T) {
				crowded := mode == "crowded" || mode == "reactor-crowded"
				bApp, ctx := setupJailGateAppWithValidators(t, 2)
				ctx = ctx.WithBlockHeight(100)
				defenderAddress, _, defenderID := newTransferPlayer(t, bApp, ctx)
				attackerAddress, _, attackerID := newTransferPlayer(t, bApp, ctx)
				validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
				require.NoError(t, err)
				denom, err := bApp.StakingKeeper.BondDenom(ctx)
				require.NoError(t, err)
				operators := make([]sdk.ValAddress, 2)
				for i, validator := range validators {
					operators[i], err = sdk.ValAddressFromBech32(validator.OperatorAddress)
					require.NoError(t, err)
				}
				delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, defenderAddress, operators[0], math.NewInt(20000000))
				delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, attackerAddress, operators[1], math.NewInt(20000000))
				if mode == "backup" {
					delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, defenderAddress, operators[1], math.NewInt(20000000))
				}
				cc := bApp.StructsKeeper.NewCurrentContext(ctx)
				const planetID, targetID, defenseID = "2-99101", "5-99101", "5-99102"
				bApp.StructsKeeper.SetPlanet(ctx, types.Planet{Id: planetID, Owner: defenderID, Creator: defenderAddress.String(),
					LandSlots: 1, Land: []string{targetID}, SpaceSlots: 1, Space: []string{defenseID}})
				cc.GetPlayer(defenderID).SetPlanetId(planetID)
				defenderFleet := cc.GetPlayer(defenderID).GetFleet().GetFleet()
				attackerFleet := cc.GetPlayer(attackerID).GetFleet().GetFleet()
				attackerStructID := attackerFleet.CommandStruct
				cc.CommitAll()
				defenderFleet.LocationId, defenderFleet.LocationType, defenderFleet.Status = planetID, types.ObjectType_planet, types.FleetStatus_onStation
				attackerFleet.LocationId, attackerFleet.LocationType, attackerFleet.Status = planetID, types.ObjectType_planet, types.FleetStatus_away
				bApp.StructsKeeper.SetFleet(ctx, defenderFleet)
				bApp.StructsKeeper.SetFleet(ctx, attackerFleet)
				defenseTypeID := uint64(19)
				if defense == "jamming" {
					defenseTypeID = 17
				}
				for _, unit := range []types.Struct{
					{Id: targetID, Index: 99101, Type: 14, Owner: defenderID, Creator: defenderAddress.String(), LocationId: planetID, LocationType: types.ObjectType_planet, OperatingAmbit: types.Ambit_land},
					{Id: defenseID, Index: 99102, Type: defenseTypeID, Owner: defenderID, Creator: defenderAddress.String(), LocationId: planetID, LocationType: types.ObjectType_planet, OperatingAmbit: types.Ambit_space},
				} {
					bApp.StructsKeeper.SetStruct(ctx, unit)
					kind, found := bApp.StructsKeeper.GetStructType(ctx, unit.Type)
					require.True(t, found)
					bApp.StructsKeeper.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, unit.Id), uint64(types.StructStateBuilt|types.StructStateMaterialized))
					bApp.StructsKeeper.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, unit.Id), kind.MaxHealth)
					unitCC := bApp.StructsKeeper.NewCurrentContext(ctx)
					unitCC.GetStruct(unit.Id).GoOnline()
					unitCC.CommitAll()
				}
				if mode == "crowded" {
					// Both owners reach their own limit. Dormant rows isolate
					// inventory work; each owner still has a live SDK-backed infusion.
					for _, id := range []string{attackerID, defenderID} {
						for i := 1; i < keeperlib.OwnerSupplyReadBudget; i++ {
							bApp.StructsKeeper.SetInfusion(ctx, types.Infusion{DestinationId: "5-99999", DestinationType: types.ObjectType_struct,
								Address: fmt.Sprintf("%s-dormant-%08d", id, i), PlayerId: id, Defusing: 1, Commission: math.LegacyZeroDec()})
						}
					}
				}


				if mode == "reactor-crowded" {
					// Every inventory row is a live SDK delegation. Distinct
					// addresses on two reactors exercise native discovery and
					// diversity, not just the cheaper dormant-generator path.
					staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
					for ownerIndex, id := range []string{attackerID, defenderID} {
						primary := []sdk.AccAddress{attackerAddress, defenderAddress}[ownerIndex]
						delegate(t, bApp, ctx, staking, denom, primary, operators[ownerIndex], math.NewInt(100))
						player, found := bApp.StructsKeeper.GetPlayer(ctx, id)
						require.True(t, found)
						for i := 1; i < keeperlib.OwnerSupplyReadBudget/2; i++ {
							bytes := make([]byte, 20)
							bytes[0], bytes[1] = 0x7e, byte(ownerIndex)
							binary.BigEndian.PutUint64(bytes[12:], uint64(i))
							address := sdk.AccAddress(bytes)
							require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, address.String(), player.Index))
							for _, operator := range operators {
								delegate(t, bApp, ctx, staking, denom, address, operator, math.NewInt(100))
							}
						}
						summary, found := bApp.StructsKeeper.GetOwnerReactorDiversity(ctx, id)
						require.True(t, found)
						require.Positive(t, summary.AppliedBonus)
					}
					require.NoError(t, bApp.StructsKeeper.CheckReactorDiversityAccounting(ctx))
				}

				cc = bApp.StructsKeeper.NewCurrentContext(ctx)
				owner := cc.GetPlayer(defenderID)
				rawCapacity := owner.GetCapacity()
				require.True(t, owner.IsOnline())
				require.True(t, bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(attackerID).IsOnline())
				if mode != "healthy" && !crowded {
					value, err := bApp.StakingKeeper.GetValidator(ctx, operators[0])
					require.NoError(t, err)
					consensus, err := value.GetConsAddr()
					require.NoError(t, err)
					require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
					if mode == "recovered" {
						require.True(t, owner.IsOffline(), "the same cached owner must observe the outage")
						require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
					}
				}
				require.Equal(t, rawCapacity, owner.GetCapacity())
				if crowded {
					// Keep fixture checks independent so the regression must
					// reach the real handler's shared-context capacity checks.
					owner = bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(defenderID)
				}
				require.Equal(t, mode != "jailed", owner.IsOnline())
				planetBefore := bApp.StructsKeeper.GetPlanetAttributesByObject(ctx, planetID)
				infusionsBefore := bApp.StructsKeeper.GetAllInfusion(ctx)
				if crowded {
					probe, _ := ctx.CacheContext()
					probe = probe.WithGasMeter(storetypes.NewInfiniteGasMeter())
					activation := bApp.StructsKeeper.NewCurrentContext(probe).GetStruct(targetID)
					activation.GoOffline()
					require.NoError(t, activation.ActivationReadinessCheck(), "the second owner-capacity check must retain its full allowance")
					t.Logf("activation readiness gas with 2048 owner infusions: %d", probe.GasMeter().GasConsumed())
					require.Less(t, probe.GasMeter().GasConsumed(), uint64(20_000_000))
				}
				for _, jamRoll := range []bool{false, true} {
					probe, _ := ctx.CacheContext()
					probe = probe.WithEventManager(sdk.NewEventManager())
					// Select each deterministic branch of the unchanged 1/3
					// single-satellite rate. No probabilities are asserted from a sample.
					var seed int64
					for seed < 100 && (rand.New(rand.NewSource(seed+1)).Intn(3) == 0) != jamRoll {
						seed++
					}
					require.Less(t, seed, int64(100))
					header := probe.BlockHeader()
					header.AppHash = make([]byte, 8)
					binary.BigEndian.PutUint64(header.AppHash, uint64(seed))
					probe = probe.WithBlockHeader(header)
					bApp.StructsKeeper.SetGridAttribute(probe, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_nonce, defenderID), 0)
					probe = probe.WithGasMeter(storetypes.NewInfiniteGasMeter())
					_, err := keeperlib.NewMsgServerImpl(bApp.StructsKeeper).StructAttack(probe, &types.MsgStructAttack{
						Creator: attackerAddress.String(), OperatingStructId: attackerStructID, TargetStructId: []string{targetID}, WeaponSystem: "primaryWeapon",
					})
					require.NoError(t, err)
					if crowded {
						t.Logf("attack gas with 2048 infusions per owner (%s): %d", mode, probe.GasMeter().GasConsumed())
						require.Less(t, probe.GasMeter().GasConsumed(), uint64(20_000_000))
					}
					var attacks []*types.EventAttack
					for _, event := range probe.EventManager().ABCIEvents() {
						if event.Type == "structs.structs.EventAttack" {
							message, err := sdk.ParseTypedEvent(event)
							require.NoError(t, err)
							attacks = append(attacks, message.(*types.EventAttack))
						}
					}
					require.Len(t, attacks, 1)
					detail := attacks[0].EventAttackDetail
					cannon := defense == "cannon" && mode != "jailed"
					jammed := defense == "jamming" && mode != "jailed" && jamRoll
					require.Equal(t, cannon, detail.PlanetaryDefenseCannonDamageToAttacker)
					expectedCannon := uint64(0)
					if cannon {
						expectedCannon = 1
					}
					require.Equal(t, expectedCannon, detail.PlanetaryDefenseCannonDamage)
					require.Equal(t, detail.AttackerHealthBefore-expectedCannon, detail.AttackerHealthAfter)
					require.Len(t, detail.EventAttackShotDetail, 1)
					shot := detail.EventAttackShotDetail[0]
					require.Equal(t, jammed, shot.EvadedByPlanetaryDefenses)
					require.False(t, shot.Evaded, "unit evasion and planetary interception are separate event fields")
					expectedDamage := uint64(2)
					if jammed {
						expectedDamage = 0
					}
					require.Equal(t, expectedDamage, shot.DamageDealt)
					check := bApp.StructsKeeper.NewCurrentContext(probe)
					require.Equal(t, detail.AttackerHealthAfter, check.GetStruct(attackerStructID).GetHealth())
					require.Equal(t, shot.TargetHealthBefore-expectedDamage, check.GetStruct(targetID).GetHealth())
					require.Equal(t, planetBefore, bApp.StructsKeeper.GetPlanetAttributesByObject(probe, planetID), "combat eligibility preserves deferred defense counters")
					require.Equal(t, infusionsBefore, bApp.StructsKeeper.GetAllInfusion(probe))
				}
			})
		}
	}
}
