package keeper_test

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructAttack(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	attackerPlayer := types.Player{
		Creator:        "cosmos1attacker",
		PrimaryAddress: "cosmos1attacker",
	}
	attackerPlayer = testAppendPlayer(k, sdkCtx, attackerPlayer)

	attackerCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attackerPlayer.Id)
	k.SetGridAttribute(sdkCtx, attackerCapAttrId, uint64(100000))
	attackerLastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, attackerPlayer.Id)
	k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))

	targetPlayer := types.Player{
		Creator:        "cosmos1target",
		PrimaryAddress: "cosmos1target",
	}
	targetPlayer = testAppendPlayer(k, sdkCtx, targetPlayer)
	targetCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, targetPlayer.Id)
	k.SetGridAttribute(sdkCtx, targetCapAttrId, uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   targetPlayer.Creator,
		Owner:     targetPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	targetPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, targetPlayer)

	cmdStructType := types.StructType{
		Id:       1,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	}
	k.SetStructType(sdkCtx, cmdStructType)

	attackStructType := types.StructType{
		Id:                   2,
		Type:                 "Gunship",
		Category:             types.ObjectType_fleet,
		PrimaryWeapon:        1,
		PrimaryWeaponCharge:  10,
		PrimaryWeaponTargets: 1,
		PrimaryWeaponAmbits:  0xFFFF,
		PrimaryWeaponDamage:  5,
		PrimaryWeaponBlockable: true,
		PossibleAmbit:        1 << uint64(types.Ambit_space),
	}
	k.SetStructType(sdkCtx, attackStructType)

	targetStructType := types.StructType{
		Id:            3,
		Type:          "Turret",
		Category:      types.ObjectType_planet,
		PossibleAmbit: 1 << uint64(types.Ambit_land),
	}
	k.SetStructType(sdkCtx, targetStructType)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      attackerPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})

	cmdStruct := types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           cmdStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	}
	cmdStruct = testAppendStruct(k, sdkCtx, cmdStruct)
	cmdStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmdStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateOnline))

	fleet.CommandStruct = cmdStruct.Id
	k.SetFleet(sdkCtx, fleet)
	attackerPlayer.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, attackerPlayer)

	attackerStruct := types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           attackStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	}
	attackerStruct = testAppendStruct(k, sdkCtx, attackerStruct)
	atkStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, attackerStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateOnline))

	targetStruct := types.Struct{
		Creator:        targetPlayer.Creator,
		Owner:          targetPlayer.Id,
		Type:           targetStructType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	}
	targetStruct = testAppendStruct(k, sdkCtx, targetStruct)
	tgtStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, targetStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateOnline))

	t.Run("valid attack", func(t *testing.T) {
		k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))
		resp, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           attackerPlayer.Creator,
			OperatingStructId: attackerStruct.Id,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           attackerPlayer.Creator,
			OperatingStructId: "invalid-struct",
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           "cosmos1noperms",
			OperatingStructId: attackerStruct.Id,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.Error(t, err)
	})
}

// TestMsgStructAttackUnbuiltTargetRejected verifies the v0.19.0 rule that a
// struct must have the Built status before it can be attacked. A target that
// is only materialized (a struct row exists but the Built flag was never set)
// must be rejected with the "unbuilt" targeting reason, regardless of its
// online/offline state.
func TestMsgStructAttackUnbuiltTargetRejected(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	attackerPlayer := types.Player{Creator: "cosmos1ubatk", PrimaryAddress: "cosmos1ubatk"}
	attackerPlayer = testAppendPlayer(k, sdkCtx, attackerPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attackerPlayer.Id), uint64(100000))
	attackerLastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, attackerPlayer.Id)
	k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))

	targetPlayer := types.Player{Creator: "cosmos1ubtgt", PrimaryAddress: "cosmos1ubtgt"}
	targetPlayer = testAppendPlayer(k, sdkCtx, targetPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, targetPlayer.Id), uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   targetPlayer.Creator,
		Owner:     targetPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	targetPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, targetPlayer)

	cmdStructType := types.StructType{Id: 500, Type: types.CommandStruct, Category: types.ObjectType_fleet}
	k.SetStructType(sdkCtx, cmdStructType)

	attackStructType := types.StructType{
		Id:                     501,
		Type:                   "Gunship",
		Category:               types.ObjectType_fleet,
		PrimaryWeapon:          1,
		PrimaryWeaponCharge:    10,
		PrimaryWeaponTargets:   1,
		PrimaryWeaponAmbits:    0xFFFF,
		PrimaryWeaponDamage:    5,
		PrimaryWeaponBlockable: true,
		PossibleAmbit:          1 << uint64(types.Ambit_space),
	}
	k.SetStructType(sdkCtx, attackStructType)

	targetStructType := types.StructType{
		Id:            502,
		Type:          "Turret",
		Category:      types.ObjectType_planet,
		PossibleAmbit: 1 << uint64(types.Ambit_land),
	}
	k.SetStructType(sdkCtx, targetStructType)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      attackerPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})

	cmdStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           cmdStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	})
	cmdStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmdStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateOnline))

	fleet.CommandStruct = cmdStruct.Id
	k.SetFleet(sdkCtx, fleet)
	attackerPlayer.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, attackerPlayer)

	attackerStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           attackStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	})
	atkStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, attackerStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateOnline))

	// Target is materialized (a struct row exists) but never receives the
	// Built flag — this is the case the v0.19.0 rule must reject.
	targetStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        targetPlayer.Creator,
		Owner:          targetPlayer.Id,
		Type:           targetStructType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	})
	tgtStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, targetStruct.Id)

	t.Run("unbuilt target rejected", func(t *testing.T) {
		k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))
		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           attackerPlayer.Creator,
			OperatingStructId: attackerStruct.Id,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.Error(t, err)

		var targetingErr *types.CombatTargetingError
		require.True(t, errors.As(err, &targetingErr), "expected a CombatTargetingError, got %T", err)
		require.Equal(t, "unbuilt", targetingErr.Reason)
	})

	t.Run("online but unbuilt target still rejected", func(t *testing.T) {
		// Online status is irrelevant: flag the target online but leave it unbuilt.
		testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateOnline))
		k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))

		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           attackerPlayer.Creator,
			OperatingStructId: attackerStruct.Id,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		require.Error(t, err)

		var targetingErr *types.CombatTargetingError
		require.True(t, errors.As(err, &targetingErr), "expected a CombatTargetingError, got %T", err)
		require.Equal(t, "unbuilt", targetingErr.Reason)
	})
}

func TestMsgStructAttackGuaranteedShots(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	attackerPlayer := types.Player{
		Creator:        "cosmos1gsattacker",
		PrimaryAddress: "cosmos1gsattacker",
	}
	attackerPlayer = testAppendPlayer(k, sdkCtx, attackerPlayer)
	attackerCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attackerPlayer.Id)
	k.SetGridAttribute(sdkCtx, attackerCapAttrId, uint64(100000))
	attackerLastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, attackerPlayer.Id)
	k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))

	targetPlayer := types.Player{
		Creator:        "cosmos1gstarget",
		PrimaryAddress: "cosmos1gstarget",
	}
	targetPlayer = testAppendPlayer(k, sdkCtx, targetPlayer)
	targetCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, targetPlayer.Id)
	k.SetGridAttribute(sdkCtx, targetCapAttrId, uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   targetPlayer.Creator,
		Owner:     targetPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	targetPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, targetPlayer)

	cmdStructType := types.StructType{
		Id:       10,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	}
	k.SetStructType(sdkCtx, cmdStructType)

	// 3 shots, 1 damage each, near-zero success rate, but 1 guaranteed shot.
	// The guaranteed shot always hits, so damage should always be >= 1.
	guaranteedShotType := types.StructType{
		Id:                                      11,
		Type:                                    "AttackRunner",
		Category:                                types.ObjectType_fleet,
		PrimaryWeapon:                           types.TechActiveWeaponry_attackRun,
		PrimaryWeaponControl:                    types.TechWeaponControl_unguided,
		PrimaryWeaponCharge:                     1,
		PrimaryWeaponTargets:                    1,
		PrimaryWeaponAmbits:                     0xFFFF,
		PrimaryWeaponShots:                      3,
		PrimaryWeaponDamage:                     1,
		PrimaryWeaponBlockable:                  true,
		PrimaryWeaponGuaranteedShots:            1,
		PrimaryWeaponShotSuccessRateNumerator:   1,
		PrimaryWeaponShotSuccessRateDenominator: 100,
		PossibleAmbit:                           1 << uint64(types.Ambit_space),
	}
	k.SetStructType(sdkCtx, guaranteedShotType)

	targetStructType := types.StructType{
		Id:            12,
		Type:          "TargetDummy",
		Category:      types.ObjectType_planet,
		MaxHealth:     10,
		PossibleAmbit: 1 << uint64(types.Ambit_land),
	}
	k.SetStructType(sdkCtx, targetStructType)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      attackerPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})

	cmdStruct := types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           cmdStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	}
	cmdStruct = testAppendStruct(k, sdkCtx, cmdStruct)
	cmdStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmdStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateOnline))

	fleet.CommandStruct = cmdStruct.Id
	k.SetFleet(sdkCtx, fleet)
	attackerPlayer.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, attackerPlayer)

	attackerStruct := types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           guaranteedShotType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	}
	attackerStruct = testAppendStruct(k, sdkCtx, attackerStruct)
	atkStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, attackerStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateOnline))

	targetStruct := types.Struct{
		Creator:        targetPlayer.Creator,
		Owner:          targetPlayer.Id,
		Type:           targetStructType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	}
	targetStruct = testAppendStruct(k, sdkCtx, targetStruct)
	tgtStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, targetStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateOnline))

	tgtHealthAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, targetStruct.Id)
	k.SetStructAttribute(sdkCtx, tgtHealthAttrId, uint64(10))

	t.Run("guaranteed shot always deals at least 1 damage", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			k.SetStructAttribute(sdkCtx, tgtHealthAttrId, uint64(10))
			testSetStructAttributeFlagRemove(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateDestroyed))
			testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateOnline))
			k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))

			resp, err := ms.StructAttack(wctx, &types.MsgStructAttack{
				Creator:           attackerPlayer.Creator,
				OperatingStructId: attackerStruct.Id,
				WeaponSystem:      "primaryWeapon",
				TargetStructId:    []string{targetStruct.Id},
			})
			require.NoError(t, err)
			require.NotNil(t, resp)

			healthAfter := k.GetStructAttribute(sdkCtx, tgtHealthAttrId)
			require.Less(t, healthAfter, uint64(10), "guaranteed shot should always deal damage (iteration %d)", i)
		}
	})
}

// TestMsgStructAttackDefenderCounterDestroysAttacker reproduces the tester
// report: a 1-HP attacker Tank attacks a Tank that is protected by another Tank
// acting as defender. The defender's same-ambit counter destroys the attacker
// before the volley can land, so neither the target nor the defender should
// take any damage. The bug was that resolveBlock consulted a stale cached
// Ready flag instead of the live destroyed/online status, so the destroyed
// attacker still inflicted blocker damage on the defender.
func TestMsgStructAttackDefenderCounterDestroysAttacker(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	// --- players ---
	atkPlayer := types.Player{Creator: "cosmos1dcatk", PrimaryAddress: "cosmos1dcatk"}
	atkPlayer = testAppendPlayer(k, sdkCtx, atkPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, atkPlayer.Id), uint64(100000))
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, atkPlayer.Id), uint64(0))

	tgtPlayer := types.Player{Creator: "cosmos1dctgt", PrimaryAddress: "cosmos1dctgt"}
	tgtPlayer = testAppendPlayer(k, sdkCtx, tgtPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, tgtPlayer.Id), uint64(100000))
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, tgtPlayer.Id), uint64(0))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   tgtPlayer.Creator,
		Owner:     tgtPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	tgtPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, tgtPlayer)

	// --- struct types ---
	cmdStructType := types.StructType{
		Id:       300,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	}
	k.SetStructType(sdkCtx, cmdStructType)

	landAmbitFlag := uint64(1) << uint64(types.Ambit_land)

	// Attacker: Tank-style, 1 HP, 2 dmg, blockable+counterable so the block
	// path is exercised in ResolveDefenders.
	atkType := types.StructType{
		Id:                                      301,
		Type:                                    "TankAttacker",
		Category:                                types.ObjectType_fleet,
		MaxHealth:                               1,
		PossibleAmbit:                           landAmbitFlag,
		PrimaryWeapon:                           types.TechActiveWeaponry_unguidedWeaponry,
		PrimaryWeaponControl:                    types.TechWeaponControl_unguided,
		PrimaryWeaponCharge:                     1,
		PrimaryWeaponTargets:                    1,
		PrimaryWeaponShots:                      1,
		PrimaryWeaponDamage:                     2,
		PrimaryWeaponAmbits:                     landAmbitFlag,
		PrimaryWeaponBlockable:                  true,
		PrimaryWeaponCounterable:                true,
		PrimaryWeaponShotSuccessRateNumerator:   1,
		PrimaryWeaponShotSuccessRateDenominator: 1,
		AttackCounterable:                       true,
	}
	k.SetStructType(sdkCtx, atkType)

	// Target: full HP, AttackReduction 1 to match the tester's Tank config.
	// No passive counter on the target itself — the only counter in the
	// scenario comes from the registered defender.
	tgtType := types.StructType{
		Id:              302,
		Type:            "TankTarget",
		Category:        types.ObjectType_planet,
		MaxHealth:       3,
		PossibleAmbit:   landAmbitFlag,
		AttackReduction: 1,
	}
	k.SetStructType(sdkCtx, tgtType)

	// Defender: full HP, same-ambit counter for 1 dmg. PrimaryWeaponAmbits is
	// required for CanCounterTargetAmbit to allow the land-ambit counter.
	defType := types.StructType{
		Id:                     303,
		Type:                   "TankDefender",
		Category:               types.ObjectType_planet,
		MaxHealth:              3,
		PossibleAmbit:          landAmbitFlag,
		PrimaryWeaponAmbits:    landAmbitFlag,
		AttackReduction:        1,
		AttackCounterable:      true,
		PassiveWeaponry:        types.TechPassiveWeaponry_counterAttack,
		CounterAttack:          1,
		CounterAttackSameAmbit: 1,
	}
	k.SetStructType(sdkCtx, defType)

	// --- attacker fleet + command struct so IsCommandable passes ---
	afleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      atkPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})
	cmd := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        atkPlayer.Creator,
		Owner:          atkPlayer.Id,
		Type:           cmdStructType.Id,
		LocationId:     afleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_land,
	})
	cmdSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmd.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdSAttr, uint64(types.StructStateOnline))
	afleet.CommandStruct = cmd.Id
	k.SetFleet(sdkCtx, afleet)
	atkPlayer.FleetId = afleet.Id
	k.SetPlayer(sdkCtx, atkPlayer)

	// Wire the planet's location list so the defender (planet-located) can
	// reach the attacker (fleet-located) via isReachable. Without this,
	// CanCounterAttack would fail with "unreachable".
	planet.LocationListStart = afleet.Id
	k.SetPlanet(sdkCtx, planet)

	// --- attacker struct, force health to 1 ---
	atkStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        atkPlayer.Creator,
		Owner:          atkPlayer.Id,
		Type:           atkType.Id,
		LocationId:     afleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_land,
	})
	atkSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, atkStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, atkSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, atkSAttr, uint64(types.StructStateOnline))
	atkHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, atkStruct.Id)
	k.SetStructAttribute(sdkCtx, atkHAttr, uint64(1))

	// --- target struct ---
	tgtStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        tgtPlayer.Creator,
		Owner:          tgtPlayer.Id,
		Type:           tgtType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	})
	tgtSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, tgtStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtSAttr, uint64(types.StructStateOnline))
	tgtHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, tgtStruct.Id)
	k.SetStructAttribute(sdkCtx, tgtHAttr, tgtType.MaxHealth)

	// --- defender struct, registered as a defender of the target ---
	defStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        tgtPlayer.Creator,
		Owner:          tgtPlayer.Id,
		Type:           defType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	})
	defSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, defStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, defSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, defSAttr, uint64(types.StructStateOnline))
	defHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, defStruct.Id)
	k.SetStructAttribute(sdkCtx, defHAttr, defType.MaxHealth)
	k.SetStructDefender(sdkCtx, tgtStruct.Id, tgtStruct.Index, defStruct.Id)

	// --- act ---
	_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
		Creator:           atkPlayer.Creator,
		OperatingStructId: atkStruct.Id,
		WeaponSystem:      "primaryWeapon",
		TargetStructId:    []string{tgtStruct.Id},
	})
	require.NoError(t, err)

	// --- assert ---
	// Attacker destroyed by the defender's counter.
	require.Equal(t, uint64(0), k.GetStructAttribute(sdkCtx, atkHAttr),
		"attacker should be at 0 HP after defender counter")
	require.True(t,
		testStructAttributeFlagHasAll(k, sdkCtx, atkSAttr, uint64(types.StructStateDestroyed)),
		"attacker should be flagged destroyed")

	// Defender takes no blocker damage from a destroyed attacker. This is the
	// assertion the bug was failing — pre-fix the defender would land on
	// MaxHealth-1.
	require.Equal(t, defType.MaxHealth, k.GetStructAttribute(sdkCtx, defHAttr),
		"defender HP should be unchanged when its counter destroyed the attacker before any block volley")

	// Target also takes no damage because the attacker is destroyed before
	// the volley would have hit it.
	require.Equal(t, tgtType.MaxHealth, k.GetStructAttribute(sdkCtx, tgtHAttr),
		"target HP should be unchanged when attacker is destroyed before volley")
}

// TestMsgStructAttackBlockerSortedBeforeLethalCounter reproduces the live
// "dead bomber deals damage" report (player 1-61, bombers 5-2116 / 5-2249): an
// air Stealth-Bomber-style attacker hits a land-operating Command Ship that is
// defended by BOTH a Tank (land — can block but cannot counter an air unit) and
// a counter unit that CAN counter air and whose counter is lethal.
//
// The defenders are iterated in byte-sorted struct-ID order, and the Tank is
// created first so it sorts ahead of the counter unit. Pre-fix, ResolveDefenders
// interleaved counter-then-block per defender, so the Tank blocked (absorbing
// the bomber's volley) while the bomber was still alive, and only the later
// counter unit destroyed it — the destroyed bomber still dealt blocker damage.
// Post-fix, all counters resolve before any block, so the lethal counter
// destroys the bomber first and no block lands: the Tank and the target take no
// damage.
func TestMsgStructAttackBlockerSortedBeforeLethalCounter(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	// --- players ---
	atkPlayer := types.Player{Creator: "cosmos1bdatk", PrimaryAddress: "cosmos1bdatk"}
	atkPlayer = testAppendPlayer(k, sdkCtx, atkPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, atkPlayer.Id), uint64(100000))
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, atkPlayer.Id), uint64(0))

	tgtPlayer := types.Player{Creator: "cosmos1bdtgt", PrimaryAddress: "cosmos1bdtgt"}
	tgtPlayer = testAppendPlayer(k, sdkCtx, tgtPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, tgtPlayer.Id), uint64(100000))
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, tgtPlayer.Id), uint64(0))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   tgtPlayer.Creator,
		Owner:     tgtPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	tgtPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, tgtPlayer)

	landAmbitFlag := uint64(1) << uint64(types.Ambit_land)
	airAmbitFlag := uint64(1) << uint64(types.Ambit_air)

	// --- struct types ---
	cmdStructType := types.StructType{
		Id:       310,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	}
	k.SetStructType(sdkCtx, cmdStructType)

	// Attacker: air Stealth-Bomber style. 3 HP, guided primary (2 dmg),
	// blockable + counterable. Primary targets land so it can hit the
	// land-operating Command Ship.
	atkType := types.StructType{
		Id:                                      311,
		Type:                                    "AirBomber",
		Category:                                types.ObjectType_fleet,
		MaxHealth:                               3,
		PossibleAmbit:                           airAmbitFlag,
		PrimaryWeapon:                           types.TechActiveWeaponry_guidedWeaponry,
		PrimaryWeaponControl:                    types.TechWeaponControl_guided,
		PrimaryWeaponCharge:                     1,
		PrimaryWeaponTargets:                    1,
		PrimaryWeaponShots:                      1,
		PrimaryWeaponDamage:                     2,
		PrimaryWeaponAmbits:                     landAmbitFlag,
		PrimaryWeaponBlockable:                  true,
		PrimaryWeaponCounterable:                true,
		PrimaryWeaponShotSuccessRateNumerator:   1,
		PrimaryWeaponShotSuccessRateDenominator: 1,
		AttackCounterable:                       true,
	}
	k.SetStructType(sdkCtx, atkType)

	// Target: land-operating Command Ship style, full HP.
	tgtType := types.StructType{
		Id:            312,
		Type:          "LandCommandShip",
		Category:      types.ObjectType_planet,
		MaxHealth:     6,
		PossibleAmbit: landAmbitFlag,
	}
	k.SetStructType(sdkCtx, tgtType)

	// Tank defender: land ambit (matches target → can block), but its weapon
	// ambits are land-only so CanCounterTargetAmbit(land, air) fails and it
	// cannot counter the air attacker. AttackReduction 1 mirrors the live Tank.
	tankType := types.StructType{
		Id:                     313,
		Type:                   "TankBlocker",
		Category:               types.ObjectType_planet,
		MaxHealth:              3,
		PossibleAmbit:          landAmbitFlag,
		PrimaryWeaponAmbits:    landAmbitFlag,
		AttackReduction:        1,
		AttackCounterable:      true,
		PassiveWeaponry:        types.TechPassiveWeaponry_counterAttack,
		CounterAttack:          1,
		CounterAttackSameAmbit: 1,
	}
	k.SetStructType(sdkCtx, tankType)

	// Counter defender: can counter air (weapon ambits include air) and its
	// counter is lethal to the 3 HP bomber in a single shot.
	counterType := types.StructType{
		Id:                     314,
		Type:                   "AirCounter",
		Category:               types.ObjectType_planet,
		MaxHealth:              3,
		PossibleAmbit:          landAmbitFlag,
		PrimaryWeaponAmbits:    airAmbitFlag,
		AttackCounterable:      true,
		PassiveWeaponry:        types.TechPassiveWeaponry_counterAttack,
		CounterAttack:          3,
		CounterAttackSameAmbit: 3,
	}
	k.SetStructType(sdkCtx, counterType)

	// --- attacker fleet + command struct so IsCommandable passes ---
	afleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      atkPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})
	cmd := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        atkPlayer.Creator,
		Owner:          atkPlayer.Id,
		Type:           cmdStructType.Id,
		LocationId:     afleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_land,
	})
	cmdSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmd.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdSAttr, uint64(types.StructStateOnline))
	afleet.CommandStruct = cmd.Id
	k.SetFleet(sdkCtx, afleet)
	atkPlayer.FleetId = afleet.Id
	k.SetPlayer(sdkCtx, atkPlayer)

	// Wire the planet's location list so the planet-located defenders can reach
	// the fleet-located attacker (isReachable), mirroring the live raid setup.
	planet.LocationListStart = afleet.Id
	k.SetPlanet(sdkCtx, planet)

	// --- attacker struct (air), force health to 3 ---
	atkStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        atkPlayer.Creator,
		Owner:          atkPlayer.Id,
		Type:           atkType.Id,
		LocationId:     afleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_air,
	})
	atkSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, atkStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, atkSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, atkSAttr, uint64(types.StructStateOnline))
	atkHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, atkStruct.Id)
	k.SetStructAttribute(sdkCtx, atkHAttr, atkType.MaxHealth)

	// --- target struct (land) ---
	tgtStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        tgtPlayer.Creator,
		Owner:          tgtPlayer.Id,
		Type:           tgtType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	})
	tgtSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, tgtStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtSAttr, uint64(types.StructStateOnline))
	tgtHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, tgtStruct.Id)
	k.SetStructAttribute(sdkCtx, tgtHAttr, tgtType.MaxHealth)

	// --- Tank defender (created FIRST so it sorts ahead of the counter unit) ---
	tankStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        tgtPlayer.Creator,
		Owner:          tgtPlayer.Id,
		Type:           tankType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	})
	tankSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, tankStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, tankSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, tankSAttr, uint64(types.StructStateOnline))
	tankHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, tankStruct.Id)
	k.SetStructAttribute(sdkCtx, tankHAttr, tankType.MaxHealth)
	k.SetStructDefender(sdkCtx, tgtStruct.Id, tgtStruct.Index, tankStruct.Id)

	// --- Counter defender (created second; lethal air counter) ---
	counterStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        tgtPlayer.Creator,
		Owner:          tgtPlayer.Id,
		Type:           counterType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	})
	counterSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, counterStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, counterSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, counterSAttr, uint64(types.StructStateOnline))
	counterHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, counterStruct.Id)
	k.SetStructAttribute(sdkCtx, counterHAttr, counterType.MaxHealth)
	k.SetStructDefender(sdkCtx, tgtStruct.Id, tgtStruct.Index, counterStruct.Id)

	// Sanity: the Tank must sort ahead of the counter unit so it is processed
	// first; this is what reproduced the bug.
	require.Less(t, tankStruct.Id, counterStruct.Id,
		"test setup expects the Tank defender to sort before the counter unit")

	// --- act ---
	_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
		Creator:           atkPlayer.Creator,
		OperatingStructId: atkStruct.Id,
		WeaponSystem:      "primaryWeapon",
		TargetStructId:    []string{tgtStruct.Id},
	})
	require.NoError(t, err)

	// --- assert ---
	// Bomber destroyed by the lethal counter.
	require.Equal(t, uint64(0), k.GetStructAttribute(sdkCtx, atkHAttr),
		"bomber should be at 0 HP after the lethal counter")
	require.True(t,
		testStructAttributeFlagHasAll(k, sdkCtx, atkSAttr, uint64(types.StructStateDestroyed)),
		"bomber should be flagged destroyed")

	// The blocking Tank takes no damage: the bomber was destroyed by counters
	// before any block could land. Pre-fix the Tank would be at MaxHealth-1.
	require.Equal(t, tankType.MaxHealth, k.GetStructAttribute(sdkCtx, tankHAttr),
		"Tank (blocker) HP should be unchanged when counters destroyed the attacker before any block")

	// The target takes no damage either.
	require.Equal(t, tgtType.MaxHealth, k.GetStructAttribute(sdkCtx, tgtHAttr),
		"target HP should be unchanged when the attacker is destroyed before its volley")
}

// TestMsgStructAttackDefenderFleetMovedNoSupport reproduces the "fleet away on
// a raid" scenario: a defender registered via SetStructDefender when its fleet
// was at the target's planet should NOT be able to counter or block once its
// fleet has moved to a different planet. Pre-fix, the block path had no range
// check at all (only ambit), so the relocated defender would still intercept
// the volley and take blocker damage. The IsProtecting filter in
// ResolveDefenders re-validates the registration-time inRange rule at action
// time and skips stale defenders.
func TestMsgStructAttackDefenderFleetMovedNoSupport(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	// --- players ---
	atkPlayer := types.Player{Creator: "cosmos1mvatk", PrimaryAddress: "cosmos1mvatk"}
	atkPlayer = testAppendPlayer(k, sdkCtx, atkPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, atkPlayer.Id), uint64(100000))
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, atkPlayer.Id), uint64(0))

	defPlayer := types.Player{Creator: "cosmos1mvdef", PrimaryAddress: "cosmos1mvdef"}
	defPlayer = testAppendPlayer(k, sdkCtx, defPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, defPlayer.Id), uint64(100000))
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, defPlayer.Id), uint64(0))

	// Target player owns the home planet (planet A) where the target lives.
	tgtPlayer := types.Player{Creator: "cosmos1mvtgt", PrimaryAddress: "cosmos1mvtgt"}
	tgtPlayer = testAppendPlayer(k, sdkCtx, tgtPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, tgtPlayer.Id), uint64(100000))

	// --- planets ---
	planetA := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator: tgtPlayer.Creator, Owner: tgtPlayer.Id, LandSlots: 4,
		Land: []string{"", "", "", ""},
	})
	tgtPlayer.PlanetId = planetA.Id
	k.SetPlayer(sdkCtx, tgtPlayer)

	planetB := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator: defPlayer.Creator, Owner: defPlayer.Id, LandSlots: 4,
		Land: []string{"", "", "", ""},
	})

	// --- struct types ---
	cmdStructType := types.StructType{
		Id:       400,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	}
	k.SetStructType(sdkCtx, cmdStructType)

	landAmbitFlag := uint64(1) << uint64(types.Ambit_land)

	// Attacker: enough HP to survive any counter (no counter actually fires
	// here, but make it large so the test is robust against unrelated changes).
	atkType := types.StructType{
		Id:                                      401,
		Type:                                    "TankAttackerMoved",
		Category:                                types.ObjectType_fleet,
		MaxHealth:                               10,
		PossibleAmbit:                           landAmbitFlag,
		PrimaryWeapon:                           types.TechActiveWeaponry_unguidedWeaponry,
		PrimaryWeaponControl:                    types.TechWeaponControl_unguided,
		PrimaryWeaponCharge:                     1,
		PrimaryWeaponTargets:                    1,
		PrimaryWeaponShots:                      1,
		PrimaryWeaponDamage:                     2,
		PrimaryWeaponAmbits:                     landAmbitFlag,
		PrimaryWeaponBlockable:                  true,
		PrimaryWeaponCounterable:                true,
		PrimaryWeaponShotSuccessRateNumerator:   1,
		PrimaryWeaponShotSuccessRateDenominator: 1,
		AttackCounterable:                       true,
	}
	k.SetStructType(sdkCtx, atkType)

	tgtType := types.StructType{
		Id:              402,
		Type:            "TankTargetMoved",
		Category:        types.ObjectType_planet,
		MaxHealth:       3,
		PossibleAmbit:   landAmbitFlag,
		AttackReduction: 1,
	}
	k.SetStructType(sdkCtx, tgtType)

	defType := types.StructType{
		Id:                     403,
		Type:                   "TankDefenderMoved",
		Category:               types.ObjectType_fleet,
		MaxHealth:              3,
		PossibleAmbit:          landAmbitFlag,
		PrimaryWeaponAmbits:    landAmbitFlag,
		AttackReduction:        1,
		AttackCounterable:      true,
		PassiveWeaponry:        types.TechPassiveWeaponry_counterAttack,
		CounterAttack:          1,
		CounterAttackSameAmbit: 1,
	}
	k.SetStructType(sdkCtx, defType)

	// --- attacker fleet + command struct, parked at planet A ---
	afleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner: atkPlayer.Id, LocationId: planetA.Id, Status: types.FleetStatus_away,
	})
	acmd := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: atkPlayer.Creator, Owner: atkPlayer.Id, Type: cmdStructType.Id,
		LocationId: afleet.Id, LocationType: types.ObjectType_fleet, OperatingAmbit: types.Ambit_land,
	})
	acmdSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, acmd.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, acmdSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, acmdSAttr, uint64(types.StructStateOnline))
	afleet.CommandStruct = acmd.Id
	k.SetFleet(sdkCtx, afleet)
	atkPlayer.FleetId = afleet.Id
	k.SetPlayer(sdkCtx, atkPlayer)

	// --- defender fleet + command struct, initially at planet A ---
	dfleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner: defPlayer.Id, LocationId: planetA.Id, Status: types.FleetStatus_away,
	})
	dcmd := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: defPlayer.Creator, Owner: defPlayer.Id, Type: cmdStructType.Id,
		LocationId: dfleet.Id, LocationType: types.ObjectType_fleet, OperatingAmbit: types.Ambit_land,
	})
	dcmdSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, dcmd.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, dcmdSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, dcmdSAttr, uint64(types.StructStateOnline))
	dfleet.CommandStruct = dcmd.Id
	k.SetFleet(sdkCtx, dfleet)
	defPlayer.FleetId = dfleet.Id
	k.SetPlayer(sdkCtx, defPlayer)

	// Planet A's fleet list head points at the attacker fleet so the attacker's
	// CanAttack passes the isReachable check from a planet-side target.
	planetA.LocationListStart = afleet.Id
	k.SetPlanet(sdkCtx, planetA)

	// --- attacker struct on attacker fleet ---
	atkStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: atkPlayer.Creator, Owner: atkPlayer.Id, Type: atkType.Id,
		LocationId: afleet.Id, LocationType: types.ObjectType_fleet, OperatingAmbit: types.Ambit_land,
	})
	atkSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, atkStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, atkSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, atkSAttr, uint64(types.StructStateOnline))
	atkHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, atkStruct.Id)
	k.SetStructAttribute(sdkCtx, atkHAttr, atkType.MaxHealth)

	// --- target struct on planet A ---
	tgtStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: tgtPlayer.Creator, Owner: tgtPlayer.Id, Type: tgtType.Id,
		LocationId: planetA.Id, LocationType: types.ObjectType_planet, OperatingAmbit: types.Ambit_land,
	})
	tgtSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, tgtStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtSAttr, uint64(types.StructStateOnline))
	tgtHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, tgtStruct.Id)
	k.SetStructAttribute(sdkCtx, tgtHAttr, tgtType.MaxHealth)

	// --- defender struct on defender fleet (initially at planet A) ---
	defStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: defPlayer.Creator, Owner: defPlayer.Id, Type: defType.Id,
		LocationId: dfleet.Id, LocationType: types.ObjectType_fleet, OperatingAmbit: types.Ambit_land,
	})
	defSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, defStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, defSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, defSAttr, uint64(types.StructStateOnline))
	defHAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, defStruct.Id)
	k.SetStructAttribute(sdkCtx, defHAttr, defType.MaxHealth)

	// Register the defender while its fleet is still at planet A.
	k.SetStructDefender(sdkCtx, tgtStruct.Id, tgtStruct.Index, defStruct.Id)

	// Now move the defender's fleet to planet B (simulating "fleet away on a
	// raid") without clearing the defender registration — that's the current
	// behavior of FleetCache.SetLocationToPlanet.
	dfleet.LocationId = planetB.Id
	k.SetFleet(sdkCtx, dfleet)

	// --- act ---
	_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
		Creator:           atkPlayer.Creator,
		OperatingStructId: atkStruct.Id,
		WeaponSystem:      "primaryWeapon",
		TargetStructId:    []string{tgtStruct.Id},
	})
	require.NoError(t, err)

	// --- assert ---
	// Defender no longer in range, so it cannot block. Pre-fix the defender
	// would have lost 1 HP from intercepting the volley (2 dmg - AttackReduction 1).
	require.Equal(t, defType.MaxHealth, k.GetStructAttribute(sdkCtx, defHAttr),
		"defender HP should be unchanged when its fleet has moved away from the protected target")

	// With no block in the way, the volley should land on the target. Pre-fix
	// the block intercepted it and the target took no damage.
	require.Less(t, k.GetStructAttribute(sdkCtx, tgtHAttr), tgtType.MaxHealth,
		"target should take volley damage when the registered defender is out of range and cannot block")

	// And the (relocated) defender cannot counter either, so the attacker is
	// unharmed.
	require.Equal(t, atkType.MaxHealth, k.GetStructAttribute(sdkCtx, atkHAttr),
		"attacker HP should be unchanged when no defender is in range to counter")
}

// TestMsgStructAttackArmourPiercing verifies the armour-piercing weapon tech:
// an armour-piercing weapon negates the target's AttackReduction entirely,
// a non-piercing weapon still has its damage reduced, and piercing against
// an unarmoured target changes nothing. The emitted EventAttack rows must
// report the piercing transparently.
func TestMsgStructAttackArmourPiercing(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)

	landAmbitFlag := uint64(1) << uint64(types.Ambit_land)

	atkPlayer := testAppendPlayer(k, sdkCtx, types.Player{
		Creator:        "cosmos1apattacker",
		PrimaryAddress: "cosmos1apattacker",
	})
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, atkPlayer.Id), uint64(100000))
	atkLastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, atkPlayer.Id)
	k.SetGridAttribute(sdkCtx, atkLastActionAttrId, uint64(0))

	tgtPlayer := testAppendPlayer(k, sdkCtx, types.Player{
		Creator:        "cosmos1aptarget",
		PrimaryAddress: "cosmos1aptarget",
	})
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, tgtPlayer.Id), uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   tgtPlayer.Creator,
		Owner:     tgtPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	tgtPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, tgtPlayer)

	cmdType := types.StructType{
		Id:       400,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	}
	k.SetStructType(sdkCtx, cmdType)

	// Armour-piercing attacker: 1 shot, 2 damage, always hits.
	apType := types.StructType{
		Id:                                      401,
		Type:                                    "PiercingGunship",
		Category:                                types.ObjectType_fleet,
		MaxHealth:                               3,
		PossibleAmbit:                           landAmbitFlag,
		PrimaryWeapon:                           types.TechActiveWeaponry_unguidedWeaponry,
		PrimaryWeaponControl:                    types.TechWeaponControl_unguided,
		PrimaryWeaponCharge:                     1,
		PrimaryWeaponTargets:                    1,
		PrimaryWeaponShots:                      1,
		PrimaryWeaponDamage:                     2,
		PrimaryWeaponAmbits:                     landAmbitFlag,
		PrimaryWeaponBlockable:                  true,
		PrimaryWeaponArmourPiercing:             true,
		PrimaryWeaponShotSuccessRateNumerator:   1,
		PrimaryWeaponShotSuccessRateDenominator: 1,
	}
	k.SetStructType(sdkCtx, apType)

	// Identical weapon without armour piercing.
	nonApType := apType
	nonApType.Id = 402
	nonApType.Type = "Gunship"
	nonApType.PrimaryWeaponArmourPiercing = false
	k.SetStructType(sdkCtx, nonApType)

	// Armoured target: AttackReduction 1, like the Tank.
	armouredType := types.StructType{
		Id:              403,
		Type:            "ArmouredTarget",
		Category:        types.ObjectType_planet,
		MaxHealth:       10,
		PossibleAmbit:   landAmbitFlag,
		UnitDefenses:    types.TechUnitDefenses_armour,
		AttackReduction: 1,
	}
	k.SetStructType(sdkCtx, armouredType)

	unarmouredType := types.StructType{
		Id:            404,
		Type:          "SoftTarget",
		Category:      types.ObjectType_planet,
		MaxHealth:     10,
		PossibleAmbit: landAmbitFlag,
	}
	k.SetStructType(sdkCtx, unarmouredType)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      atkPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})
	cmd := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: atkPlayer.Creator, Owner: atkPlayer.Id, Type: cmdType.Id,
		LocationId: fleet.Id, LocationType: types.ObjectType_fleet, OperatingAmbit: types.Ambit_land,
	})
	cmdSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmd.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdSAttr, uint64(types.StructStateOnline))
	fleet.CommandStruct = cmd.Id
	k.SetFleet(sdkCtx, fleet)
	atkPlayer.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, atkPlayer)

	makeStruct := func(player types.Player, typeId uint64, locationId string, locationType types.ObjectType) (types.Struct, string) {
		structure := testAppendStruct(k, sdkCtx, types.Struct{
			Creator: player.Creator, Owner: player.Id, Type: typeId,
			LocationId: locationId, LocationType: locationType, OperatingAmbit: types.Ambit_land,
		})
		sAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id)
		testSetStructAttributeFlagAdd(k, sdkCtx, sAttr, uint64(types.StructStateBuilt))
		testSetStructAttributeFlagAdd(k, sdkCtx, sAttr, uint64(types.StructStateOnline))
		hAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, structure.Id)
		return structure, hAttr
	}

	apAttacker, _ := makeStruct(atkPlayer, apType.Id, fleet.Id, types.ObjectType_fleet)
	nonApAttacker, _ := makeStruct(atkPlayer, nonApType.Id, fleet.Id, types.ObjectType_fleet)
	armouredTarget, armouredHAttr := makeStruct(tgtPlayer, armouredType.Id, planet.Id, types.ObjectType_planet)
	unarmouredTarget, unarmouredHAttr := makeStruct(tgtPlayer, unarmouredType.Id, planet.Id, types.ObjectType_planet)

	// attack runs one StructAttack with a fresh event manager and returns the
	// emitted EventAttack for shot-detail assertions.
	attack := func(t *testing.T, attackerId string, targetId string) *types.EventAttack {
		t.Helper()
		k.SetGridAttribute(sdkCtx, atkLastActionAttrId, uint64(0))
		eventCtx := sdkCtx.WithEventManager(sdk.NewEventManager())

		_, err := ms.StructAttack(sdk.WrapSDKContext(eventCtx), &types.MsgStructAttack{
			Creator:           atkPlayer.Creator,
			OperatingStructId: attackerId,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetId},
		})
		require.NoError(t, err)

		for _, abciEvent := range eventCtx.EventManager().ABCIEvents() {
			if abciEvent.Type != "structs.structs.EventAttack" {
				continue
			}
			msg, err := sdk.ParseTypedEvent(abciEvent)
			require.NoError(t, err)
			attackEvent, ok := msg.(*types.EventAttack)
			require.True(t, ok)
			return attackEvent
		}
		t.Fatal("no EventAttack emitted")
		return nil
	}

	t.Run("armour piercing negates attack reduction", func(t *testing.T) {
		k.SetStructAttribute(sdkCtx, armouredHAttr, armouredType.MaxHealth)

		attackEvent := attack(t, apAttacker.Id, armouredTarget.Id)

		require.Equal(t, armouredType.MaxHealth-2, k.GetStructAttribute(sdkCtx, armouredHAttr),
			"armour-piercing volley should land full damage")

		shots := attackEvent.EventAttackDetail.EventAttackShotDetail
		require.Len(t, shots, 1)
		require.True(t, shots[0].ArmourPiercing, "event must report armour piercing")
		require.Equal(t, uint64(0), shots[0].DamageReduction, "no reduction applied when pierced")
		require.Equal(t, types.TechUnitDefenses_armour, shots[0].DamageReductionCause, "event must report what was pierced")
		require.Equal(t, uint64(2), shots[0].Damage)
	})

	t.Run("non-piercing weapon is still reduced by armour", func(t *testing.T) {
		k.SetStructAttribute(sdkCtx, armouredHAttr, armouredType.MaxHealth)

		attackEvent := attack(t, nonApAttacker.Id, armouredTarget.Id)

		require.Equal(t, armouredType.MaxHealth-1, k.GetStructAttribute(sdkCtx, armouredHAttr),
			"non-piercing volley should be reduced by armour")

		shots := attackEvent.EventAttackDetail.EventAttackShotDetail
		require.Len(t, shots, 1)
		require.False(t, shots[0].ArmourPiercing)
		require.Equal(t, uint64(1), shots[0].DamageReduction)
		require.Equal(t, types.TechUnitDefenses_armour, shots[0].DamageReductionCause)
		require.Equal(t, uint64(1), shots[0].Damage)
	})

	t.Run("armour piercing against unarmoured target changes nothing", func(t *testing.T) {
		k.SetStructAttribute(sdkCtx, unarmouredHAttr, unarmouredType.MaxHealth)

		attackEvent := attack(t, apAttacker.Id, unarmouredTarget.Id)

		require.Equal(t, unarmouredType.MaxHealth-2, k.GetStructAttribute(sdkCtx, unarmouredHAttr))

		shots := attackEvent.EventAttackDetail.EventAttackShotDetail
		require.Len(t, shots, 1)
		require.False(t, shots[0].ArmourPiercing, "no piercing reported when target has no reduction")
		require.Equal(t, uint64(0), shots[0].DamageReduction)
		require.Equal(t, uint64(2), shots[0].Damage)
	})
}

// TestMsgStructAttackCommandShipNotRequired verifies the v0.19.0 change that an
// attack no longer requires the fleet's Command Ship to be present or online.
// The attacking struct itself must still be online; only the command-ship gate
// was removed.
func TestMsgStructAttackCommandShipNotRequired(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	attackerPlayer := types.Player{Creator: "cosmos1ncatk", PrimaryAddress: "cosmos1ncatk"}
	attackerPlayer = testAppendPlayer(k, sdkCtx, attackerPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attackerPlayer.Id), uint64(100000))
	attackerLastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, attackerPlayer.Id)
	k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))

	targetPlayer := types.Player{Creator: "cosmos1nctgt", PrimaryAddress: "cosmos1nctgt"}
	targetPlayer = testAppendPlayer(k, sdkCtx, targetPlayer)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, targetPlayer.Id), uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   targetPlayer.Creator,
		Owner:     targetPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	targetPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, targetPlayer)

	cmdStructType := types.StructType{Id: 600, Type: types.CommandStruct, Category: types.ObjectType_fleet}
	k.SetStructType(sdkCtx, cmdStructType)

	attackStructType := types.StructType{
		Id:                     601,
		Type:                   "Gunship",
		Category:               types.ObjectType_fleet,
		PrimaryWeapon:          1,
		PrimaryWeaponCharge:    10,
		PrimaryWeaponTargets:   1,
		PrimaryWeaponAmbits:    0xFFFF,
		PrimaryWeaponDamage:    5,
		PrimaryWeaponBlockable: true,
		PossibleAmbit:          1 << uint64(types.Ambit_space),
	}
	k.SetStructType(sdkCtx, attackStructType)

	targetStructType := types.StructType{
		Id:            602,
		Type:          "Turret",
		Category:      types.ObjectType_planet,
		MaxHealth:     30,
		PossibleAmbit: 1 << uint64(types.Ambit_land),
	}
	k.SetStructType(sdkCtx, targetStructType)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      attackerPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})

	cmdStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           cmdStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	})
	cmdStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmdStruct.Id)
	// Command ship is built but deliberately NOT online.
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateBuilt))

	fleet.CommandStruct = cmdStruct.Id
	k.SetFleet(sdkCtx, fleet)
	attackerPlayer.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, attackerPlayer)

	attackerStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        attackerPlayer.Creator,
		Owner:          attackerPlayer.Id,
		Type:           attackStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	})
	atkStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, attackerStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, atkStatusAttrId, uint64(types.StructStateOnline))

	targetStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        targetPlayer.Creator,
		Owner:          targetPlayer.Id,
		Type:           targetStructType.Id,
		LocationId:     planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
	})
	tgtStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, targetStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, tgtStatusAttrId, uint64(types.StructStateOnline))

	attack := func(t *testing.T) error {
		t.Helper()
		k.SetGridAttribute(sdkCtx, attackerLastActionAttrId, uint64(0))
		_, err := ms.StructAttack(wctx, &types.MsgStructAttack{
			Creator:           attackerPlayer.Creator,
			OperatingStructId: attackerStruct.Id,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetStruct.Id},
		})
		return err
	}

	t.Run("attack succeeds with command ship offline", func(t *testing.T) {
		require.NoError(t, attack(t))
	})

	t.Run("attack succeeds with command ship destroyed", func(t *testing.T) {
		testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateDestroyed))
		require.NoError(t, attack(t))
	})

	t.Run("attack succeeds with no command struct on the fleet", func(t *testing.T) {
		fleet.CommandStruct = ""
		k.SetFleet(sdkCtx, fleet)
		require.NoError(t, attack(t))
	})
}

// TestMsgStructAttackPlanetaryDefenseGuidedOnly verifies the Jamming Satellite
// planetary defense (lowOrbitBallisticInterceptorNetwork). The defense shields
// a planetary struct from guided ordnance regardless of source/target ambit and
// never shields fleet-located structs. It covers:
//   - a guided weapon is jammed (fully evaded);
//   - an otherwise identical unguided weapon passes through and lands damage;
//   - ambit-independence: a guided attack is jammed even when attacker and
//     target both operate in space (excluded under the old air/space -> land/water rule);
//   - a fleet-located target is never shielded, even for a guided weapon;
//   - with no active interceptor network, a guided attack lands normally.
//
// The planet's interceptor success rate is forced to 1/1 so IsSuccessful is
// deterministic: a guided shot always evades, and passing shots are proven to
// bypass the planetary defense entirely rather than merely winning a roll.
// Targets carry no unit defenses, so the only possible evasion is planetary.
func TestMsgStructAttackPlanetaryDefenseGuidedOnly(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)

	atkPlayer := testAppendPlayer(k, sdkCtx, types.Player{
		Creator:        "cosmos1pdatk",
		PrimaryAddress: "cosmos1pdatk",
	})
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, atkPlayer.Id), uint64(100000))
	atkLastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, atkPlayer.Id)
	k.SetGridAttribute(sdkCtx, atkLastActionAttrId, uint64(0))

	tgtPlayer := testAppendPlayer(k, sdkCtx, types.Player{
		Creator:        "cosmos1pdtgt",
		PrimaryAddress: "cosmos1pdtgt",
	})
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, tgtPlayer.Id), uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   tgtPlayer.Creator,
		Owner:     tgtPlayer.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	tgtPlayer.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, tgtPlayer)

	// Force a guaranteed interceptor success (1/1) so evasion is deterministic.
	k.SetPlanetAttribute(sdkCtx, keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator, planet.Id), uint64(1))
	k.SetPlanetAttribute(sdkCtx, keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator, planet.Id), uint64(1))

	landAmbitFlag := uint64(1) << uint64(types.Ambit_land)

	cmdType := types.StructType{Id: 700, Type: types.CommandStruct, Category: types.ObjectType_fleet}
	k.SetStructType(sdkCtx, cmdType)

	// Guided attacker: space fleet struct that can target land, 1 shot, 3 damage.
	guidedType := types.StructType{
		Id:                                      701,
		Type:                                    "GuidedGunship",
		Category:                                types.ObjectType_fleet,
		MaxHealth:                               3,
		PossibleAmbit:                           1 << uint64(types.Ambit_space),
		PrimaryWeapon:                           types.TechActiveWeaponry_guidedWeaponry,
		PrimaryWeaponControl:                    types.TechWeaponControl_guided,
		PrimaryWeaponCharge:                     1,
		PrimaryWeaponTargets:                    1,
		PrimaryWeaponShots:                      1,
		PrimaryWeaponDamage:                     3,
		PrimaryWeaponAmbits:                     landAmbitFlag,
		PrimaryWeaponBlockable:                  true,
		PrimaryWeaponShotSuccessRateNumerator:   1,
		PrimaryWeaponShotSuccessRateDenominator: 1,
	}
	k.SetStructType(sdkCtx, guidedType)

	// Unguided attacker: identical except for weapon control.
	unguidedType := guidedType
	unguidedType.Id = 702
	unguidedType.Type = "UnguidedGunship"
	unguidedType.PrimaryWeapon = types.TechActiveWeaponry_unguidedWeaponry
	unguidedType.PrimaryWeaponControl = types.TechWeaponControl_unguided
	k.SetStructType(sdkCtx, unguidedType)

	// Target: land planetary struct with no unit defenses.
	targetType := types.StructType{
		Id:            703,
		Type:          "GroundTarget",
		Category:      types.ObjectType_planet,
		MaxHealth:     10,
		PossibleAmbit: landAmbitFlag,
	}
	k.SetStructType(sdkCtx, targetType)

	spaceAmbitFlag := uint64(1) << uint64(types.Ambit_space)

	// Guided attacker whose weapon reaches the space ambit. Used to prove the
	// planetary defense no longer depends on source/target ambit: under the old
	// rules a space attacker striking a space target was excluded.
	spaceGuidedType := guidedType
	spaceGuidedType.Id = 704
	spaceGuidedType.Type = "SpaceGuidedGunship"
	spaceGuidedType.PrimaryWeaponAmbits = spaceAmbitFlag
	k.SetStructType(sdkCtx, spaceGuidedType)

	// Planetary target operating in space. The old ambit rules only jammed
	// water/land targets, so this target proves ambit-independence.
	spaceTargetType := targetType
	spaceTargetType.Id = 705
	spaceTargetType.Type = "SpaceGroundTarget"
	spaceTargetType.PossibleAmbit = spaceAmbitFlag
	k.SetStructType(sdkCtx, spaceTargetType)

	// Fleet-located target: planetary defenses must never shield a fleet struct.
	fleetTargetType := targetType
	fleetTargetType.Id = 706
	fleetTargetType.Type = "FleetTarget"
	fleetTargetType.Category = types.ObjectType_fleet
	fleetTargetType.PossibleAmbit = spaceAmbitFlag
	k.SetStructType(sdkCtx, fleetTargetType)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      atkPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})
	cmd := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: atkPlayer.Creator, Owner: atkPlayer.Id, Type: cmdType.Id,
		LocationId: fleet.Id, LocationType: types.ObjectType_fleet, OperatingAmbit: types.Ambit_space,
	})
	cmdSAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmd.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdSAttr, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdSAttr, uint64(types.StructStateOnline))
	fleet.CommandStruct = cmd.Id
	k.SetFleet(sdkCtx, fleet)
	atkPlayer.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, atkPlayer)

	makeStruct := func(player types.Player, typeId uint64, locationId string, locationType types.ObjectType, ambit types.Ambit) (types.Struct, string) {
		structure := testAppendStruct(k, sdkCtx, types.Struct{
			Creator: player.Creator, Owner: player.Id, Type: typeId,
			LocationId: locationId, LocationType: locationType, OperatingAmbit: ambit,
		})
		sAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id)
		testSetStructAttributeFlagAdd(k, sdkCtx, sAttr, uint64(types.StructStateBuilt))
		testSetStructAttributeFlagAdd(k, sdkCtx, sAttr, uint64(types.StructStateOnline))
		hAttr := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, structure.Id)
		return structure, hAttr
	}

	guidedAttacker, _ := makeStruct(atkPlayer, guidedType.Id, fleet.Id, types.ObjectType_fleet, types.Ambit_space)
	unguidedAttacker, _ := makeStruct(atkPlayer, unguidedType.Id, fleet.Id, types.ObjectType_fleet, types.Ambit_space)
	spaceGuidedAttacker, _ := makeStruct(atkPlayer, spaceGuidedType.Id, fleet.Id, types.ObjectType_fleet, types.Ambit_space)
	target, targetHAttr := makeStruct(tgtPlayer, targetType.Id, planet.Id, types.ObjectType_planet, types.Ambit_land)
	spaceTarget, spaceTargetHAttr := makeStruct(tgtPlayer, spaceTargetType.Id, planet.Id, types.ObjectType_planet, types.Ambit_space)

	// Fleet-located target sits in the defender's own fleet docked at the planet
	// so it is reachable, but is not a planetary struct. The struct cache
	// resolves a fleet struct's planet via its owner's FleetId, so wire it up.
	tgtFleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      tgtPlayer.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})
	tgtPlayer.FleetId = tgtFleet.Id
	k.SetPlayer(sdkCtx, tgtPlayer)
	fleetTarget, fleetTargetHAttr := makeStruct(tgtPlayer, fleetTargetType.Id, tgtFleet.Id, types.ObjectType_fleet, types.Ambit_space)

	// attack runs one StructAttack with a fresh event manager and returns the
	// emitted EventAttack for shot-detail assertions.
	attack := func(t *testing.T, attackerId string, targetId string) *types.EventAttack {
		t.Helper()
		k.SetGridAttribute(sdkCtx, atkLastActionAttrId, uint64(0))
		eventCtx := sdkCtx.WithEventManager(sdk.NewEventManager())
		_, err := ms.StructAttack(sdk.WrapSDKContext(eventCtx), &types.MsgStructAttack{
			Creator:           atkPlayer.Creator,
			OperatingStructId: attackerId,
			WeaponSystem:      "primaryWeapon",
			TargetStructId:    []string{targetId},
		})
		require.NoError(t, err)

		for _, abciEvent := range eventCtx.EventManager().ABCIEvents() {
			if abciEvent.Type != "structs.structs.EventAttack" {
				continue
			}
			msg, err := sdk.ParseTypedEvent(abciEvent)
			require.NoError(t, err)
			attackEvent, ok := msg.(*types.EventAttack)
			require.True(t, ok)
			return attackEvent
		}
		t.Fatal("no EventAttack emitted")
		return nil
	}

	t.Run("guided weapon is jammed by the planetary defense", func(t *testing.T) {
		k.SetStructAttribute(sdkCtx, targetHAttr, targetType.MaxHealth)

		attackEvent := attack(t, guidedAttacker.Id, target.Id)

		require.Equal(t, targetType.MaxHealth, k.GetStructAttribute(sdkCtx, targetHAttr),
			"guided attack must be fully evaded by the planetary defense")

		shots := attackEvent.EventAttackDetail.EventAttackShotDetail
		require.Len(t, shots, 1)
		require.True(t, shots[0].EvadedByPlanetaryDefenses, "guided shot must be evaded by planetary defense")
		require.Equal(t, types.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork, shots[0].EvadedByPlanetaryDefensesCause)
	})

	t.Run("unguided weapon bypasses the planetary defense", func(t *testing.T) {
		k.SetStructAttribute(sdkCtx, targetHAttr, targetType.MaxHealth)

		attackEvent := attack(t, unguidedAttacker.Id, target.Id)

		require.Equal(t, targetType.MaxHealth-3, k.GetStructAttribute(sdkCtx, targetHAttr),
			"unguided attack must land full damage (not jammed)")

		shots := attackEvent.EventAttackDetail.EventAttackShotDetail
		require.Len(t, shots, 1)
		require.False(t, shots[0].EvadedByPlanetaryDefenses, "unguided shot must not be evaded by planetary defense")
	})

	// Ambit-independence: a guided attack on a planetary target is jammed even
	// when source and target share the space ambit, which the old rule excluded.
	t.Run("guided weapon is jammed regardless of ambit", func(t *testing.T) {
		k.SetStructAttribute(sdkCtx, spaceTargetHAttr, spaceTargetType.MaxHealth)

		attackEvent := attack(t, spaceGuidedAttacker.Id, spaceTarget.Id)

		require.Equal(t, spaceTargetType.MaxHealth, k.GetStructAttribute(sdkCtx, spaceTargetHAttr),
			"guided attack on a planetary target must be jammed even when both structs operate in space")

		shots := attackEvent.EventAttackDetail.EventAttackShotDetail
		require.Len(t, shots, 1)
		require.True(t, shots[0].EvadedByPlanetaryDefenses, "guided shot must be evaded regardless of ambit")
		require.Equal(t, types.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork, shots[0].EvadedByPlanetaryDefensesCause)
	})

	// Planetary-target requirement: a fleet-located target is never shielded by
	// the planetary defense, even for guided weapons on a defended planet.
	t.Run("fleet-located target is not shielded by the planetary defense", func(t *testing.T) {
		k.SetStructAttribute(sdkCtx, fleetTargetHAttr, fleetTargetType.MaxHealth)

		attackEvent := attack(t, spaceGuidedAttacker.Id, fleetTarget.Id)

		require.Equal(t, fleetTargetType.MaxHealth-3, k.GetStructAttribute(sdkCtx, fleetTargetHAttr),
			"guided attack on a fleet target must land full damage; only planetary structs are shielded")

		shots := attackEvent.EventAttackDetail.EventAttackShotDetail
		require.Len(t, shots, 1)
		require.False(t, shots[0].EvadedByPlanetaryDefenses, "fleet target must not be evaded by planetary defense")
	})

	// Active-network requirement: with no interceptor network on the planet, a
	// guided attack on a planetary target lands normally.
	t.Run("guided weapon is not evaded when no satellite is active", func(t *testing.T) {
		numId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator, planet.Id)
		denomId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator, planet.Id)
		k.SetPlanetAttribute(sdkCtx, numId, uint64(0))
		k.SetPlanetAttribute(sdkCtx, denomId, uint64(0))
		t.Cleanup(func() {
			k.SetPlanetAttribute(sdkCtx, numId, uint64(1))
			k.SetPlanetAttribute(sdkCtx, denomId, uint64(1))
		})

		k.SetStructAttribute(sdkCtx, targetHAttr, targetType.MaxHealth)

		attackEvent := attack(t, guidedAttacker.Id, target.Id)

		require.Equal(t, targetType.MaxHealth-3, k.GetStructAttribute(sdkCtx, targetHAttr),
			"guided attack must land when the planet has no active interceptor network")

		shots := attackEvent.EventAttackDetail.EventAttackShotDetail
		require.Len(t, shots, 1)
		require.False(t, shots[0].EvadedByPlanetaryDefenses, "shot must not be evaded when no satellite is active")
	})
}
