package v0_18_0_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/app/upgrades"
	v0_18_0 "structs/app/upgrades/v0_18_0"
	keepertest "structs/testutil/keeper"
	structskeeper "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// --- local fixtures -------------------------------------------------------

func appendPlayer(k structskeeper.Keeper, ctx context.Context, address string) types.Player {
	count := k.GetPlayerCount(ctx)
	player := types.Player{
		Creator:        address,
		PrimaryAddress: address,
		Index:          count,
		Id:             fmt.Sprintf("%d-%d", types.ObjectType_player, count),
	}
	k.SetPlayer(ctx, player)
	k.SetPlayerCount(ctx, count+1)
	return player
}

func appendPlanet(k structskeeper.Keeper, ctx context.Context, planet types.Planet) types.Planet {
	count := k.GetPlanetCount(ctx)
	planet.Id = fmt.Sprintf("%d-%d", types.ObjectType_planet, count)
	if planet.Status == 0 {
		planet.Status = types.PlanetStatus_active
	}
	k.SetPlanetCount(ctx, count+1)
	k.SetPlanet(ctx, planet)
	return planet
}

func appendStruct(k structskeeper.Keeper, ctx context.Context, structure types.Struct, status types.StructState) types.Struct {
	count := k.GetStructCount(ctx)
	structure.Index = count
	structure.Id = fmt.Sprintf("%d-%d", types.ObjectType_struct, count)
	k.SetStructCount(ctx, count+1)
	k.SetStruct(ctx, structure)

	statusAttrId := structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id)
	k.SetStructAttribute(ctx, statusAttrId, uint64(status))
	return structure
}

func appendFleet(k structskeeper.Keeper, ctx context.Context, fleet types.Fleet) types.Fleet {
	ownerIndex := "0"
	if fleet.Owner != "" {
		ownerIndex = fleet.Owner[len(fleet.Owner)-1:]
	}
	fleet.Id = fmt.Sprintf("%d-%s", types.ObjectType_fleet, ownerIndex)
	k.SetFleet(ctx, fleet)
	return fleet
}

func shieldAttrId(planetId string) string {
	return structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, planetId)
}

func raidAttrId(planetId string) string {
	return structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, planetId)
}

func healthAttrId(structId string) string {
	return structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_health, structId)
}

// --- tests ----------------------------------------------------------------

// TestMigrateStructTypes_RebasesShieldContributions verifies the rebased
// PlanetaryShieldContribution values land in the struct type store.
func TestMigrateStructTypes_RebasesShieldContributions(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	// Seed stale pre-upgrade values so the rewrite is observable.
	k.SetStructType(ctx, types.StructType{Id: 16, PlanetaryShieldContribution: 1500, BuildLimit: 1})
	k.SetStructType(ctx, types.StructType{Id: 18, PlanetaryShieldContribution: 9000, BuildLimit: 1})

	require.NoError(t, v0_18_0.MigrateStructTypes(ctx, keepers))

	expected := map[uint64]uint64{
		16: 25, // Orbital Shield Generator
		17: 12, // Jamming Satellite
		18: 50, // Ore Bunker
		19: 13, // Planetary Defense Cannon
	}
	for typeId, contribution := range expected {
		structType, found := k.GetStructType(ctx, typeId)
		require.True(t, found, "struct type %d must exist after rewrite", typeId)
		require.Equal(t, contribution, structType.PlanetaryShieldContribution, "struct type %d contribution", typeId)
	}

	// The pure-shield defense structs are now unlimited (BuildLimit 0) so
	// players can stack them to extend the shields-vulnerable window.
	for _, typeId := range []uint64{16, 18} {
		structType, found := k.GetStructType(ctx, typeId)
		require.True(t, found, "struct type %d must exist after rewrite", typeId)
		require.Equal(t, uint64(0), structType.BuildLimit, "struct type %d build limit", typeId)
	}
}

// TestMigrateStructTypes_RebalancesBattleship verifies the Battleship vs.
// Tank balancing lands in the struct type store: armour-piercing unguided
// primary restricted to land+water, and a new guided space secondary.
func TestMigrateStructTypes_RebalancesBattleship(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	// Seed a stale pre-upgrade Battleship so the rewrite is observable.
	k.SetStructType(ctx, types.StructType{
		Id:                  2,
		PrimaryWeaponAmbits: 22,
		SecondaryWeapon:     types.TechActiveWeaponry_noActiveWeaponry,
	})

	require.NoError(t, v0_18_0.MigrateStructTypes(ctx, keepers))

	battleship, found := k.GetStructType(ctx, 2)
	require.True(t, found, "Battleship struct type must exist after rewrite")

	// Primary: armour-piercing unguided, land + water only.
	require.Equal(t, uint64(6), battleship.PrimaryWeaponAmbits, "primary ambits = water + land")
	require.True(t, battleship.PrimaryWeaponArmourPiercing, "primary is armour piercing")
	require.Equal(t, types.TechWeaponControl_unguided, battleship.PrimaryWeaponControl)

	// Secondary: guided space weapon.
	require.Equal(t, types.TechActiveWeaponry_guidedWeaponry, battleship.SecondaryWeapon)
	require.Equal(t, types.TechWeaponControl_guided, battleship.SecondaryWeaponControl)
	require.Equal(t, uint64(16), battleship.SecondaryWeaponAmbits, "secondary ambits = space")
	require.Equal(t, uint64(8), battleship.SecondaryWeaponCharge)
	require.Equal(t, uint64(1), battleship.SecondaryWeaponDamage)
	require.False(t, battleship.SecondaryWeaponArmourPiercing)

	// Tank keeps its armour; no struct type other than the Battleship
	// primary gains armour piercing.
	tank, found := k.GetStructType(ctx, 9)
	require.True(t, found)
	require.Equal(t, uint64(1), tank.AttackReduction, "tank retains attack reduction")
	for _, structType := range types.CreateStructTypeGenesis() {
		if structType.Id == 2 {
			continue
		}
		require.False(t, structType.PrimaryWeaponArmourPiercing, "type %d primary must not be armour piercing", structType.Id)
		require.False(t, structType.SecondaryWeaponArmourPiercing, "type %d secondary must not be armour piercing", structType.Id)
	}
}

// TestMigrateStructTypes_RaisesPlanetaryHealth verifies the planetary
// max-health rebase lands in the struct type store: baseline planetary
// structs go to 6, the three power generators are hardened to 8/10/10
// with armour + attackReduction 1, and fleet struct types are untouched.
func TestMigrateStructTypes_RaisesPlanetaryHealth(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	// Seed stale pre-upgrade health so the rewrite is observable.
	for _, typeId := range []uint64{14, 15, 16, 17, 18, 19, 20, 21, 22} {
		k.SetStructType(ctx, types.StructType{Id: typeId, MaxHealth: 3})
	}

	require.NoError(t, v0_18_0.MigrateStructTypes(ctx, keepers))

	expectedHealth := map[uint64]uint64{
		14: 6, 15: 6, 16: 6, 17: 6, 18: 6, 19: 6, // baseline planetary
		20: 8, 21: 10, 22: 10, // power generators
	}
	for typeId, health := range expectedHealth {
		structType, found := k.GetStructType(ctx, typeId)
		require.True(t, found, "struct type %d must exist after rewrite", typeId)
		require.Equal(t, health, structType.MaxHealth, "struct type %d max health", typeId)
	}

	// Power generators gain armour + attackReduction 1 (the Tank's pairing).
	for _, typeId := range []uint64{20, 21, 22} {
		structType, _ := k.GetStructType(ctx, typeId)
		require.Equal(t, types.TechUnitDefenses_armour, structType.UnitDefenses, "struct type %d unit defenses", typeId)
		require.Equal(t, uint64(1), structType.AttackReduction, "struct type %d attack reduction", typeId)
	}

	// Fleet struct types keep their 3 max health.
	for _, typeId := range []uint64{2, 3, 8, 9} {
		structType, found := k.GetStructType(ctx, typeId)
		require.True(t, found)
		require.Equal(t, uint64(3), structType.MaxHealth, "fleet struct type %d max health unchanged", typeId)
	}
}

// TestMigratePlanetaryStructHealth verifies built, non-destroyed planetary
// structs are raised to their new MaxHealth, while destroyed planetary
// structs, unbuilt structs, and fleet structs are left untouched.
func TestMigratePlanetaryStructHealth(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	require.NoError(t, v0_18_0.MigrateStructTypes(ctx, keepers))

	planet := appendPlanet(k, ctx, types.Planet{Owner: "1-1"})

	online := types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline
	offlineBuilt := types.StructStateMaterialized | types.StructStateBuilt
	building := types.StructStateMaterialized
	destroyed := types.StructStateMaterialized | types.StructStateBuilt | types.StructStateDestroyed

	// Damaged online PDC (type 19, new max 6).
	damagedPDC := appendStruct(k, ctx, types.Struct{Type: 19, LocationType: types.ObjectType_planet, LocationId: planet.Id}, online)
	k.SetStructAttribute(ctx, healthAttrId(damagedPDC.Id), 1)

	// Offline-but-built Field Generator (type 20, new max 8) sitting at old health.
	offlineGen := appendStruct(k, ctx, types.Struct{Type: 20, LocationType: types.ObjectType_planet, LocationId: planet.Id}, offlineBuilt)
	k.SetStructAttribute(ctx, healthAttrId(offlineGen.Id), 3)

	// World Engine (type 22, new max 10) at full old health.
	worldEngine := appendStruct(k, ctx, types.Struct{Type: 22, LocationType: types.ObjectType_planet, LocationId: planet.Id}, online)
	k.SetStructAttribute(ctx, healthAttrId(worldEngine.Id), 3)

	// Still-building planetary struct must not be touched (not yet built).
	unbuilt := appendStruct(k, ctx, types.Struct{Type: 19, LocationType: types.ObjectType_planet, LocationId: planet.Id}, building)
	k.SetStructAttribute(ctx, healthAttrId(unbuilt.Id), 3)

	// Destroyed planetary struct must not be revived.
	deadBunker := appendStruct(k, ctx, types.Struct{Type: 18, LocationType: types.ObjectType_planet, LocationId: planet.Id}, destroyed)
	k.SetStructAttribute(ctx, healthAttrId(deadBunker.Id), 0)

	// Fleet struct (Tank, type 9) must not be touched.
	fleetTank := appendStruct(k, ctx, types.Struct{Type: 9, LocationType: types.ObjectType_fleet, LocationId: "9-1"}, online)
	k.SetStructAttribute(ctx, healthAttrId(fleetTank.Id), 2)

	require.NoError(t, v0_18_0.MigratePlanetaryStructHealth(ctx, keepers))

	require.Equal(t, uint64(6), k.GetStructAttribute(ctx, healthAttrId(damagedPDC.Id)), "damaged online PDC raised to new max")
	require.Equal(t, uint64(8), k.GetStructAttribute(ctx, healthAttrId(offlineGen.Id)), "offline-but-built generator raised to new max")
	require.Equal(t, uint64(10), k.GetStructAttribute(ctx, healthAttrId(worldEngine.Id)), "world engine raised to new max")
	require.Equal(t, uint64(3), k.GetStructAttribute(ctx, healthAttrId(unbuilt.Id)), "unbuilt struct untouched")
	require.Equal(t, uint64(0), k.GetStructAttribute(ctx, healthAttrId(deadBunker.Id)), "destroyed struct untouched")
	require.Equal(t, uint64(2), k.GetStructAttribute(ctx, healthAttrId(fleetTank.Id)), "fleet struct untouched")

	// Idempotent: a re-run produces the same values.
	require.NoError(t, v0_18_0.MigratePlanetaryStructHealth(ctx, keepers))
	require.Equal(t, uint64(6), k.GetStructAttribute(ctx, healthAttrId(damagedPDC.Id)))
	require.Equal(t, uint64(10), k.GetStructAttribute(ctx, healthAttrId(worldEngine.Id)))
}

// TestMigratePlanetaryShields_RecomputesFromOnlineDefenses verifies the
// two-pass shield rebase: every planet drops to the new base, and online
// defense structs add their new (not old) contributions.
func TestMigratePlanetaryShields_RecomputesFromOnlineDefenses(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	require.NoError(t, v0_18_0.MigrateStructTypes(ctx, keepers))

	// Planet with no defense structs: stale hour-scale shield value.
	barePlanet := appendPlanet(k, ctx, types.Planet{Owner: "1-1"})
	k.SetPlanetAttribute(ctx, shieldAttrId(barePlanet.Id), uint64(1500))

	// Planet with an online Ore Bunker (50), an online PDC (13), an
	// offline Orbital Shield Generator (must not count), and an online
	// non-defense Ore Extractor (must not count).
	defendedPlanet := appendPlanet(k, ctx, types.Planet{Owner: "1-2"})
	k.SetPlanetAttribute(ctx, shieldAttrId(defendedPlanet.Id), uint64(21000))

	online := types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline
	offline := types.StructStateMaterialized | types.StructStateBuilt

	appendStruct(k, ctx, types.Struct{Type: 18, LocationType: types.ObjectType_planet, LocationId: defendedPlanet.Id}, online)
	appendStruct(k, ctx, types.Struct{Type: 19, LocationType: types.ObjectType_planet, LocationId: defendedPlanet.Id}, online)
	appendStruct(k, ctx, types.Struct{Type: 16, LocationType: types.ObjectType_planet, LocationId: defendedPlanet.Id}, offline)
	appendStruct(k, ctx, types.Struct{Type: 14, LocationType: types.ObjectType_planet, LocationId: defendedPlanet.Id}, online)

	// Online defense struct on a fleet (not a planet) must not count.
	appendStruct(k, ctx, types.Struct{Type: 17, LocationType: types.ObjectType_fleet, LocationId: "9-1"}, online)

	require.NoError(t, v0_18_0.MigratePlanetaryShields(ctx, keepers))

	require.Equal(t, uint64(types.PlanetaryShieldBase), k.GetPlanetAttribute(ctx, shieldAttrId(barePlanet.Id)), "bare planet rebased to new base")
	require.Equal(t, uint64(types.PlanetaryShieldBase+50+13), k.GetPlanetAttribute(ctx, shieldAttrId(defendedPlanet.Id)), "defended planet = base + online contributions")

	// Idempotent: a re-run recomputes the same values.
	require.NoError(t, v0_18_0.MigratePlanetaryShields(ctx, keepers))
	require.Equal(t, uint64(types.PlanetaryShieldBase+50+13), k.GetPlanetAttribute(ctx, shieldAttrId(defendedPlanet.Id)))
}

// TestMigrateBlockStartRaid_NormalizesRaidClocks verifies the blockStartRaid
// normalization across the four defender states: no raid, raid vs online
// Command Ship, raid vs offline Command Ship, and raid vs missing fleet.
func TestMigrateBlockStartRaid_NormalizesRaidClocks(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	upgradeHeight := int64(777_000)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(upgradeHeight)

	online := types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline
	offline := types.StructStateMaterialized | types.StructStateBuilt

	// Helper: player + fleet + command ship in the given status.
	makeDefender := func(addrSeed string, cmdStatus types.StructState, withFleet bool) types.Player {
		player := appendPlayer(k, sdkCtx, "cosmos1"+addrSeed+strconv.FormatUint(k.GetPlayerCount(sdkCtx), 10))
		if !withFleet {
			return player
		}
		commandShip := appendStruct(k, sdkCtx, types.Struct{Owner: player.Id, Type: types.CommandStructTypeId, LocationType: types.ObjectType_fleet}, cmdStatus)
		fleet := appendFleet(k, sdkCtx, types.Fleet{Owner: player.Id, CommandStruct: commandShip.Id})
		player.FleetId = fleet.Id
		k.SetPlayer(sdkCtx, player)
		return player
	}

	// (a) No raid in progress: stale clock must be cleared.
	idle := makeDefender("idle", online, true)
	idlePlanet := appendPlanet(k, sdkCtx, types.Planet{Owner: idle.Id})
	k.SetPlanetAttribute(sdkCtx, raidAttrId(idlePlanet.Id), uint64(12345))

	// (b) Raid in progress, defender Command Ship online: cleared.
	shielded := makeDefender("shielded", online, true)
	shieldedPlanet := appendPlanet(k, sdkCtx, types.Planet{Owner: shielded.Id, LocationListStart: "9-900"})
	k.SetPlanetAttribute(sdkCtx, raidAttrId(shieldedPlanet.Id), uint64(12345))

	// (c) Raid in progress, defender Command Ship offline: re-anchored to
	// the upgrade height.
	vulnerable := makeDefender("vulnerable", offline, true)
	vulnerablePlanet := appendPlanet(k, sdkCtx, types.Planet{Owner: vulnerable.Id, LocationListStart: "9-901"})
	k.SetPlanetAttribute(sdkCtx, raidAttrId(vulnerablePlanet.Id), uint64(12345))

	// (d) Raid in progress, defender has no fleet at all: re-anchored.
	fleetless := makeDefender("fleetless", online, false)
	fleetlessPlanet := appendPlanet(k, sdkCtx, types.Planet{Owner: fleetless.Id, LocationListStart: "9-902"})
	k.SetPlanetAttribute(sdkCtx, raidAttrId(fleetlessPlanet.Id), uint64(12345))

	require.NoError(t, v0_18_0.MigrateBlockStartRaid(sdkCtx, keepers))

	require.Equal(t, uint64(0), k.GetPlanetAttribute(sdkCtx, raidAttrId(idlePlanet.Id)), "no raid: clock cleared")
	require.Equal(t, uint64(0), k.GetPlanetAttribute(sdkCtx, raidAttrId(shieldedPlanet.Id)), "raid vs online command ship: clock cleared")
	require.Equal(t, uint64(upgradeHeight), k.GetPlanetAttribute(sdkCtx, raidAttrId(vulnerablePlanet.Id)), "raid vs offline command ship: re-anchored")
	require.Equal(t, uint64(upgradeHeight), k.GetPlanetAttribute(sdkCtx, raidAttrId(fleetlessPlanet.Id)), "raid vs missing fleet: re-anchored")

	// Idempotent at the same height: identical rows.
	require.NoError(t, v0_18_0.MigrateBlockStartRaid(sdkCtx, keepers))
	require.Equal(t, uint64(upgradeHeight), k.GetPlanetAttribute(sdkCtx, raidAttrId(vulnerablePlanet.Id)))
}
