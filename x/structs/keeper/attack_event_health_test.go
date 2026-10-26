package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// HealthBefore is an observation, not HealthAfter plus nominal volley damage:
// damage saturates at zero and a target can be visited more than once per message.
func TestMsgStructAttackEventRecordsObservedTargetHealth(t *testing.T) {
	for _, tc := range []struct {
		name     string
		health   []uint64
		shots    uint64
		repeated bool
		blocked  bool
		evaded   bool
		rebound  bool
	}{
		{name: "single shot overkill", health: []uint64{1}, shots: 1},
		{name: "single shot survivor", health: []uint64{5}, shots: 1},
		{name: "multiple projectiles overkill", health: []uint64{1}, shots: 3},
		{name: "distinct targets preserve their own health", health: []uint64{1, 5}, shots: 1},
		{name: "multiple targets and projectiles", health: []uint64{1, 5}, shots: 3},
		{name: "repeated target starts from previous volley", health: []uint64{5, 3}, shots: 1, repeated: true},
		{name: "blocker overkill leaves target untouched", health: []uint64{5}, shots: 1, blocked: true},
		{name: "multi projectile blocker leaves target untouched", health: []uint64{5}, shots: 2, blocked: true},
		{name: "evasion leaves target untouched", health: []uint64{1}, shots: 1, evaded: true},
		{name: "target counter triggers lethal attacker explosion", health: []uint64{3}, shots: 1, rebound: true},
		{name: "multi projectile target counter triggers explosion", health: []uint64{5}, shots: 2, rebound: true},
		{name: "evading target dies to attacker explosion", health: []uint64{1}, shots: 1, evaded: true, rebound: true},
		{name: "blocked target dies to attacker explosion", health: []uint64{1}, shots: 1, blocked: true, rebound: true},
		{name: "multi projectile blocked target dies to explosion", health: []uint64{1}, shots: 2, blocked: true, rebound: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ms, ctx := setupMsgServer(t)
			sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1000).WithEventManager(sdk.NewEventManager())
			attackerPlayer := testAppendPlayer(k, sdkCtx, types.Player{Creator: "cosmos1healthatk", PrimaryAddress: "cosmos1healthatk"})
			targetPlayer := testAppendPlayer(k, sdkCtx, types.Player{Creator: "cosmos1healthtgt", PrimaryAddress: "cosmos1healthtgt"})
			for _, player := range []types.Player{attackerPlayer, targetPlayer} {
				k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), 10000000)
				k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id), 0)
			}
			planet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: targetPlayer.Creator, Owner: targetPlayer.Id, LandSlots: 4, Land: []string{"", "", "", ""}})
			targetPlayer.PlanetId = planet.Id
			k.SetPlayer(sdkCtx, targetPlayer)
			fleet := testAppendFleet(k, sdkCtx, types.Fleet{Owner: attackerPlayer.Id, LocationId: planet.Id, Status: types.FleetStatus_away})
			attackerPlayer.FleetId = fleet.Id
			k.SetPlayer(sdkCtx, attackerPlayer)
			planet.LocationListStart = fleet.Id
			k.SetPlanet(sdkCtx, planet)

			land := uint64(1) << uint64(types.Ambit_land)
			attackerType := types.StructType{
				Id: 800, Type: "Health event attacker", Category: types.ObjectType_fleet, MaxHealth: 6, PossibleAmbit: land,
				PrimaryWeapon: types.TechActiveWeaponry_unguidedWeaponry, PrimaryWeaponControl: types.TechWeaponControl_unguided,
				PrimaryWeaponTargets: uint64(len(tc.health)), PrimaryWeaponShots: tc.shots, PrimaryWeaponDamage: 2,
				PrimaryWeaponAmbits: land, PrimaryWeaponBlockable: true, PrimaryWeaponCharge: 1,
				PrimaryWeaponShotSuccessRateNumerator: 1, PrimaryWeaponShotSuccessRateDenominator: 1,
			}
			targetType := types.StructType{Id: 801, Type: "Health event target", Category: types.ObjectType_planet, MaxHealth: 6, PossibleAmbit: land}
			attackerHealth := uint64(6)
			if tc.rebound {
				attackerHealth = 1
				attackerType.AttackCounterable = true
				attackerType.PrimaryWeaponCounterable = true
				attackerType.PostDestructionDamage = 2
				targetType.PrimaryWeaponAmbits = land
				targetType.CounterAttack = 1
				targetType.CounterAttackSameAmbit = 1
				targetType.PassiveWeaponry = types.TechPassiveWeaponry_counterAttack
			}
			if tc.evaded {
				targetType.UnitDefenses = types.TechUnitDefenses_defensiveManeuver
				targetType.UnguidedDefensiveSuccessRateNumerator = 1
				targetType.UnguidedDefensiveSuccessRateDenominator = 1
			}
			defenderType := types.StructType{Id: 802, Type: "Health event blocker", Category: types.ObjectType_planet, MaxHealth: 3, PossibleAmbit: land, CanDefend: true}
			for _, kind := range []types.StructType{attackerType, targetType, defenderType} {
				k.SetStructType(sdkCtx, kind)
			}
			add := func(player types.Player, kind types.StructType, location string, health uint64) types.Struct {
				structure := testAppendStruct(k, sdkCtx, types.Struct{Creator: player.Creator, Owner: player.Id, Type: kind.Id, LocationId: location, LocationType: kind.Category, OperatingAmbit: types.Ambit_land})
				status := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id)
				testSetStructAttributeFlagAdd(k, sdkCtx, status, uint64(types.StructStateBuilt|types.StructStateOnline))
				k.SetStructAttribute(sdkCtx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, structure.Id), health)
				return structure
			}
			attacker := add(attackerPlayer, attackerType, fleet.Id, attackerHealth)
			var targets []string
			for i, health := range tc.health {
				if i > 0 && tc.repeated {
					targets = append(targets, targets[0])
					continue
				}
				target := add(targetPlayer, targetType, planet.Id, health)
				targets = append(targets, target.Id)
				if tc.blocked {
					defender := add(targetPlayer, defenderType, planet.Id, 1)
					k.SetStructDefender(sdkCtx, target.Id, target.Index, defender.Id)
				}
			}

			_, err := ms.StructAttack(sdk.WrapSDKContext(sdkCtx), &types.MsgStructAttack{Creator: attackerPlayer.Creator, OperatingStructId: attacker.Id, WeaponSystem: "primaryWeapon", TargetStructId: targets})
			require.NoError(t, err)
			var attacks []*types.EventAttack
			for _, event := range sdkCtx.EventManager().ABCIEvents() {
				if event.Type == "structs.structs.EventAttack" {
					message, parseErr := sdk.ParseTypedEvent(event)
					require.NoError(t, parseErr)
					attacks = append(attacks, message.(*types.EventAttack))
				}
			}
			require.Len(t, attacks, 1)
			detail := attacks[0].EventAttackDetail
			require.Equal(t, attackerHealth, detail.AttackerHealthBefore)
			if tc.rebound {
				require.Zero(t, detail.AttackerHealthAfter)
			} else {
				require.Equal(t, attackerHealth, detail.AttackerHealthAfter)
			}
			require.Len(t, detail.EventAttackShotDetail, len(targets)*int(tc.shots))
			for i, target := range targets {
				remaining := uint64(0)
				if tc.blocked || tc.evaded {
					remaining = tc.health[i]
				} else if tc.health[i] > 2*tc.shots {
					remaining = tc.health[i] - 2*tc.shots
				}
				if tc.rebound {
					remaining = 0
				}
				for projectile := uint64(0); projectile < tc.shots; projectile++ {
					row := detail.EventAttackShotDetail[i*int(tc.shots)+int(projectile)]
					require.Equal(t, target, row.TargetStructId)
					require.Equal(t, tc.health[i], row.TargetHealthBefore, "event must retain actual pre-volley health, including overkill")
					require.Equal(t, tc.blocked, row.Blocked)
					require.Equal(t, tc.evaded, row.Evaded)
					if projectile == tc.shots-1 {
						if tc.rebound {
							require.True(t, row.TargetCountered)
							require.True(t, row.TargetCounterDestroyedAttacker)
						}
						require.Equal(t, remaining, row.TargetHealthAfter)
						require.Equal(t, remaining == 0, row.TargetDestroyed)
						if !tc.evaded {
							require.Equal(t, 2*tc.shots, row.Damage, "nominal damage remains separate from health actually lost")
						}
						if tc.blocked {
							require.Equal(t, uint64(1), row.BlockerHealthBefore)
							require.Zero(t, row.BlockerHealthAfter)
							require.True(t, row.BlockerDestroyed)
						}
					} else {
						require.Equal(t, tc.health[i], row.TargetHealthAfter)
					}
				}
				if !tc.repeated || i == len(targets)-1 {
					health := k.GetStructAttribute(sdkCtx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, target))
					require.Equal(t, remaining, health, "canonical damage remains correct")
				}
			}
		})
	}
}
