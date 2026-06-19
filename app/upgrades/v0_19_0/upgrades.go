package v0_19_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
	structskeeper "structs/x/structs/keeper"
	structstypes "structs/x/structs/types"
)

// CreateUpgradeHandler returns the v0.19.0 upgrade handler.
//
// Beyond the binary-only raid handler changes described in constants.go,
// this upgrade carries two one-time state migrations at upgrade height:
//
//   - MigrateStructTypes rewrites every struct type from
//     CreateStructTypeGenesis(), applying the Battleship secondary weapon
//     damage rebalance (1 -> 2). Struct types are stored in state, so the
//     genesis change only reaches the live chain through this rewrite.
//
//   - MigrateAwayDefenderRaidClock backfills blockStartRaid for in-progress
//     raids that became winnable under the broadened vulnerability predicate
//     (defending fleet away) but were left with a zero clock by v0.18.0.
//
// Both migrations are idempotent: each is a pure projection of the current
// state (genesis constants; raid queue + defending fleet location + Command
// Ship status), so a re-run (e.g. a validator replaying the upgrade block
// from a snapshot) produces the same rows.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *upgrades.Keepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		newVM, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return newVM, err
		}

		if err := MigrateStructTypes(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateAwayDefenderRaidClock(ctx, keepers); err != nil {
			return newVM, err
		}

		return newVM, nil
	}
}

// MigrateStructTypes rewrites all struct types from the genesis definitions,
// applying any struct type tuning shipped with this release (the Battleship
// secondary weapon damage 1 -> 2 rebalance). Struct types are persisted in
// state, so this rewrite is how the genesis change reaches an already-live
// chain. Idempotent: it is a pure projection of the genesis constants.
func MigrateStructTypes(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateStructTypes")

	structTypes := structstypes.CreateStructTypeGenesis()
	for _, structType := range structTypes {
		keepers.StructsKeeper.SetStructType(ctx, structType)
	}

	logger.Info("v0.19.0 struct-type rewrite complete", "structTypesWritten", len(structTypes))
	return nil
}

// MigrateAwayDefenderRaidClock anchors the blockStartRaid vulnerability
// window for in-progress raids that are vulnerable under the v0.19.0
// predicate but still have a zero clock.
//
// v0.18.0 only considered the Command Ship's online/destroyed status, so a
// raid against a defender who had moved their fleet (and Command Ship) away
// was treated as shielded and its clock cleared. Under v0.19.0 that raid is
// vulnerable, but MsgPlanetRaidComplete rejects a zero clock
// (raid_clock_unset), leaving it permanently uncompletable until some event
// restarts the clock. This stamps such raids to the upgrade height.
//
// The v0.19.0 predicate is strictly broader than v0.18.0's (it only adds the
// defender-away case), so no planet becomes less vulnerable. Raids that
// already have a running clock (offline/destroyed Command Ship) must NOT be
// re-anchored, so only still-zero clocks are filled. Idempotent: a re-run
// only fills clocks that are still zero.
func MigrateAwayDefenderRaidClock(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateAwayDefenderRaidClock")

	k := keepers.StructsKeeper
	upgradeHeight := uint64(sdkCtx.BlockHeight())

	var raidsAnchored int
	for _, planet := range k.GetAllPlanet(ctx) {
		if planet.LocationListStart == "" {
			continue
		}

		raidAttributeId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_blockStartRaid, planet.Id)

		// Leave running clocks (offline/destroyed Command Ship) untouched.
		if k.GetPlanetAttribute(ctx, raidAttributeId) != 0 {
			continue
		}

		if !isDefenderCommandStructVulnerable(ctx, keepers, planet.Owner) {
			continue
		}

		k.SetPlanetAttribute(ctx, raidAttributeId, upgradeHeight)
		raidsAnchored++
		logger.Info("anchored away-defender raid vulnerability window", "planetId", planet.Id, "raidFleetId", planet.LocationListStart, "blockStartRaid", upgradeHeight)
	}

	logger.Info("v0.19.0 away-defender raid clock backfill complete", "raidsAnchored", raidsAnchored)
	return nil
}

// isDefenderCommandStructVulnerable mirrors
// PlanetCache.IsDefenderCommandStructVulnerable using raw keeper reads (no
// caches, no auto-creation side effects): the defending player's Command
// Ship is absent (fleet not on station), offline, destroyed, or
// non-existent.
func isDefenderCommandStructVulnerable(ctx context.Context, keepers *upgrades.Keepers, ownerId string) bool {
	k := keepers.StructsKeeper

	if ownerId == "" {
		return true
	}

	player, playerFound := k.GetPlayer(ctx, ownerId)
	if !playerFound || player.FleetId == "" {
		return true
	}

	fleet, fleetFound := k.GetFleet(ctx, player.FleetId)
	if !fleetFound || fleet.CommandStruct == "" {
		return true
	}

	// Command Ship only defends the home planet while the fleet is on station
	if fleet.Status != structstypes.FleetStatus_onStation {
		return true
	}

	statusAttributeId := structskeeper.GetStructAttributeIDByObjectId(structstypes.StructAttributeType_status, fleet.CommandStruct)
	status := structstypes.StructState(k.GetStructAttribute(ctx, statusAttributeId))

	if status&structstypes.StructStateDestroyed != 0 {
		return true
	}

	return status&structstypes.StructStateOnline == 0
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
