package v0_19_0_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/app/upgrades"
	v0_19_0 "structs/app/upgrades/v0_19_0"
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

func raidAttrId(planetId string) string {
	return structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, planetId)
}

// --- tests ----------------------------------------------------------------

// TestMigrateAwayDefenderRaidClock verifies the v0.19.0 backfill: an
// in-progress raid against a defender whose fleet is away (and whose clock
// was cleared by v0.18.0) is anchored to the upgrade height, while running
// clocks, shielded raids, and idle planets are left untouched.
func TestMigrateAwayDefenderRaidClock(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	upgradeHeight := int64(888_000)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(upgradeHeight)

	online := types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline
	offline := types.StructStateMaterialized | types.StructStateBuilt

	// Helper: player + fleet (in the given location status) + command ship.
	makeDefender := func(addrSeed string, cmdStatus types.StructState, fleetStatus types.FleetStatus, withFleet bool) types.Player {
		player := appendPlayer(k, sdkCtx, "cosmos1"+addrSeed+strconv.FormatUint(k.GetPlayerCount(sdkCtx), 10))
		if !withFleet {
			return player
		}
		commandShip := appendStruct(k, sdkCtx, types.Struct{Owner: player.Id, Type: types.CommandStructTypeId, LocationType: types.ObjectType_fleet}, cmdStatus)
		fleet := appendFleet(k, sdkCtx, types.Fleet{Owner: player.Id, CommandStruct: commandShip.Id, Status: fleetStatus})
		player.FleetId = fleet.Id
		k.SetPlayer(sdkCtx, player)
		return player
	}

	// (a) Raid in progress, defender fleet away with an ONLINE command ship,
	// clock cleared by v0.18.0: this is the case the backfill targets.
	away := makeDefender("away", online, types.FleetStatus_away, true)
	awayPlanet := appendPlanet(k, sdkCtx, types.Planet{Owner: away.Id, LocationListStart: "9-901"})
	k.SetPlanetAttribute(sdkCtx, raidAttrId(awayPlanet.Id), uint64(0))

	// (b) Raid in progress, defender fleet on station with an online command
	// ship: still shielded, must stay at zero.
	shielded := makeDefender("shielded", online, types.FleetStatus_onStation, true)
	shieldedPlanet := appendPlanet(k, sdkCtx, types.Planet{Owner: shielded.Id, LocationListStart: "9-902"})
	k.SetPlanetAttribute(sdkCtx, raidAttrId(shieldedPlanet.Id), uint64(0))

	// (c) Raid in progress, defender command ship offline with a running
	// clock: must NOT be re-anchored.
	running := makeDefender("running", offline, types.FleetStatus_onStation, true)
	runningPlanet := appendPlanet(k, sdkCtx, types.Planet{Owner: running.Id, LocationListStart: "9-903"})
	k.SetPlanetAttribute(sdkCtx, raidAttrId(runningPlanet.Id), uint64(42))

	// (d) No raid in progress, defender away: nothing to anchor.
	idle := makeDefender("idle", online, types.FleetStatus_away, true)
	idlePlanet := appendPlanet(k, sdkCtx, types.Planet{Owner: idle.Id})
	k.SetPlanetAttribute(sdkCtx, raidAttrId(idlePlanet.Id), uint64(0))

	require.NoError(t, v0_19_0.MigrateAwayDefenderRaidClock(sdkCtx, keepers))

	require.Equal(t, uint64(upgradeHeight), k.GetPlanetAttribute(sdkCtx, raidAttrId(awayPlanet.Id)), "raid vs away defender: anchored to upgrade height")
	require.Equal(t, uint64(0), k.GetPlanetAttribute(sdkCtx, raidAttrId(shieldedPlanet.Id)), "raid vs on-station online command ship: left shielded")
	require.Equal(t, uint64(42), k.GetPlanetAttribute(sdkCtx, raidAttrId(runningPlanet.Id)), "running clock left untouched")
	require.Equal(t, uint64(0), k.GetPlanetAttribute(sdkCtx, raidAttrId(idlePlanet.Id)), "no raid: left at zero")

	// Idempotent at the same height: identical rows.
	require.NoError(t, v0_19_0.MigrateAwayDefenderRaidClock(sdkCtx, keepers))
	require.Equal(t, uint64(upgradeHeight), k.GetPlanetAttribute(sdkCtx, raidAttrId(awayPlanet.Id)))
	require.Equal(t, uint64(42), k.GetPlanetAttribute(sdkCtx, raidAttrId(runningPlanet.Id)))
}

// TestMigrateStructTypes_RaisesBattleshipSecondaryDamage verifies the
// Battleship secondary weapon damage rebalance (1 -> 2) lands in the struct
// type store, overwriting a stale pre-upgrade value.
func TestMigrateStructTypes_RaisesBattleshipSecondaryDamage(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	// Seed a stale pre-upgrade Battleship so the rewrite is observable.
	k.SetStructType(ctx, types.StructType{Id: 2, SecondaryWeaponDamage: 1})

	require.NoError(t, v0_19_0.MigrateStructTypes(ctx, keepers))

	battleship, found := k.GetStructType(ctx, 2)
	require.True(t, found, "Battleship struct type must exist after rewrite")
	require.Equal(t, uint64(2), battleship.SecondaryWeaponDamage, "battleship secondary weapon damage raised to 2")

	// Idempotent: a re-run produces the same value.
	require.NoError(t, v0_19_0.MigrateStructTypes(ctx, keepers))
	battleship, _ = k.GetStructType(ctx, 2)
	require.Equal(t, uint64(2), battleship.SecondaryWeaponDamage)
}
