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
