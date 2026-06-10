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

// --- tests ----------------------------------------------------------------

// TestMigrateStructTypes_RebasesShieldContributions verifies the rebased
// PlanetaryShieldContribution values land in the struct type store.
func TestMigrateStructTypes_RebasesShieldContributions(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	// Seed stale pre-upgrade values so the rewrite is observable.
	k.SetStructType(ctx, types.StructType{Id: 16, PlanetaryShieldContribution: 1500})
	k.SetStructType(ctx, types.StructType{Id: 18, PlanetaryShieldContribution: 9000})

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
