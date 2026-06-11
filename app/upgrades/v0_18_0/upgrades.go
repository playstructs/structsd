package v0_18_0

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

// CreateUpgradeHandler returns the v0.18.0 upgrade handler.
//
// Beyond the binary-only raid handler changes described in constants.go,
// this upgrade carries three one-time state migrations at upgrade height,
// in this order (the ordering matters: the shield recompute reads the new
// PlanetaryShieldContribution values written by the struct-type rewrite):
//
//  1. MigrateStructTypes rewrites every struct type from
//     CreateStructTypeGenesis(), applying the rebased planetary shield
//     contributions and the raised planetary max-health values.
//
//  2. MigratePlanetaryStructHealth sets every built, non-destroyed
//     planetary struct's health to its (newly raised) type MaxHealth.
//     Health is only stamped at materialization, so without this
//     pre-upgrade planetary structs would keep their old 3 health against
//     the new 6/8/10 maxima and appear permanently damaged.
//
//  3. MigratePlanetaryShields recomputes every planet's planetaryShield
//     attribute as the new base plus the new contributions of its online
//     defense structs. Without this, pre-upgrade planets would keep
//     hour-scale shield values (1500-21000) while GoOffline decrements
//     only the new small contributions, permanently corrupting shields.
//
//  4. MigrateBlockStartRaid normalizes every planet's blockStartRaid
//     attribute to the new Command-Ship-driven semantics: upgrade height
//     where a raid is in progress against a vulnerable defender, zero
//     otherwise. A stale value would either collapse the puzzle
//     difficulty to trivial or anchor it to a dead hash input.
//
// All four migrations are idempotent: each is a pure projection of the
// current state (genesis constants, online defense structs, raid queue +
// Command Ship status) rather than an incremental adjustment, so a re-run
// (e.g. a validator replaying the upgrade block from a snapshot) produces
// the same rows. A re-run of MigrateBlockStartRaid at a later height
// re-anchors in-progress vulnerability windows to that height, which is
// safe: the raid clock restarts and proofs ground against the old input
// are simply invalidated.
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

		if err := MigratePlanetaryStructHealth(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigratePlanetaryShields(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateBlockStartRaid(ctx, keepers); err != nil {
			return newVM, err
		}

		return newVM, nil
	}
}

// MigrateStructTypes rewrites all struct types from the genesis
// definitions, applying the rebased PlanetaryShieldContribution values
// (and any other struct type tuning shipped with this release).
func MigrateStructTypes(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateStructTypes")

	structTypes := structstypes.CreateStructTypeGenesis()
	for _, structType := range structTypes {
		keepers.StructsKeeper.SetStructType(ctx, structType)
	}

	logger.Info("v0.18.0 struct-type rewrite complete", "structTypesWritten", len(structTypes))
	return nil
}

// MigratePlanetaryStructHealth raises the stored health of every built,
// non-destroyed planetary struct to its (newly raised) type MaxHealth.
//
// Struct health is only stamped to MaxHealth at materialization; build
// completion and going online never touch it. Bumping the struct type's
// MaxHealth therefore leaves pre-upgrade planetary structs sitting at
// their old health (3) against the new maxima (6/8/10), where they would
// read as permanently damaged with no in-game way to repair. This sets
// each one to full health once, at upgrade height.
//
// Must run after MigrateStructTypes so it reads the raised MaxHealth.
// It is O(structs) with a memoized struct-type cache, mirroring
// MigratePlanetaryShields. Idempotent: a pure projection to MaxHealth.
func MigratePlanetaryStructHealth(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migratePlanetaryStructHealth")

	k := keepers.StructsKeeper

	// Struct types are few; memoize to avoid a store read per struct.
	structTypeCache := make(map[uint64]structstypes.StructType)

	var healthsRaised int
	for _, structure := range k.GetAllStruct(ctx) {
		if structure.LocationType != structstypes.ObjectType_planet {
			continue
		}

		statusAttributeId := structskeeper.GetStructAttributeIDByObjectId(structstypes.StructAttributeType_status, structure.Id)
		status := structstypes.StructState(k.GetStructAttribute(ctx, statusAttributeId))
		if status&structstypes.StructStateBuilt == 0 || status&structstypes.StructStateDestroyed != 0 {
			continue
		}

		structType, cached := structTypeCache[structure.Type]
		if !cached {
			loadedType, found := k.GetStructType(ctx, structure.Type)
			if !found {
				logger.Warn("struct references unknown struct type; skipping", "structId", structure.Id, "structType", structure.Type)
				continue
			}
			structType = loadedType
			structTypeCache[structure.Type] = structType
		}

		healthAttributeId := structskeeper.GetStructAttributeIDByObjectId(structstypes.StructAttributeType_health, structure.Id)
		k.SetStructAttribute(ctx, healthAttributeId, structType.MaxHealth)
		healthsRaised++
	}

	logger.Info("v0.18.0 planetary struct-health rebase complete", "planetaryStructsRaised", healthsRaised)
	return nil
}

// MigratePlanetaryShields recomputes every planet's planetaryShield
// attribute under the rebased difficulty scale in two passes:
//
//   - Pass 1 sets every planet's shield to the new PlanetaryShieldBase.
//   - Pass 2 walks all structs once and increments the owning planet's
//     shield for each online defense struct.
//
// This is O(planets + structs); a per-planet struct lookup would be
// O(planets x structs) and needlessly stretch upgrade-height downtime on
// a chain with many depleted planets.
func MigratePlanetaryShields(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migratePlanetaryShields")

	k := keepers.StructsKeeper

	allPlanets := k.GetAllPlanet(ctx)
	for _, planet := range allPlanets {
		shieldAttributeId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_planetaryShield, planet.Id)
		k.SetPlanetAttribute(ctx, shieldAttributeId, structstypes.PlanetaryShieldBase)
	}

	// Struct types are few; memoize to avoid a store read per struct.
	structTypeCache := make(map[uint64]structstypes.StructType)

	var contributionsApplied int
	for _, structure := range k.GetAllStruct(ctx) {
		if structure.LocationType != structstypes.ObjectType_planet {
			continue
		}

		structType, cached := structTypeCache[structure.Type]
		if !cached {
			loadedType, found := k.GetStructType(ctx, structure.Type)
			if !found {
				logger.Warn("struct references unknown struct type; skipping", "structId", structure.Id, "structType", structure.Type)
				continue
			}
			structType = loadedType
			structTypeCache[structure.Type] = structType
		}

		if !structType.HasOreReserveDefensesSystem() || structType.PlanetaryShieldContribution == 0 {
			continue
		}

		statusAttributeId := structskeeper.GetStructAttributeIDByObjectId(structstypes.StructAttributeType_status, structure.Id)
		status := structstypes.StructState(k.GetStructAttribute(ctx, statusAttributeId))
		if status&structstypes.StructStateOnline == 0 {
			continue
		}

		shieldAttributeId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_planetaryShield, structure.LocationId)
		k.SetPlanetAttribute(ctx, shieldAttributeId, k.GetPlanetAttribute(ctx, shieldAttributeId)+structType.PlanetaryShieldContribution)
		contributionsApplied++
	}

	logger.Info("v0.18.0 planetary-shield rebase complete", "planetsRebased", len(allPlanets), "onlineDefenseContributions", contributionsApplied)
	return nil
}

// MigrateBlockStartRaid normalizes every planet's blockStartRaid attribute
// to the new Command-Ship-driven semantics:
//
//   - A raid is in progress (locationListStart != "") and the defending
//     Command Ship is offline, destroyed, or non-existent: the
//     vulnerability window is anchored at the upgrade height.
//   - Otherwise: cleared to zero. The handler-side guards reject raid
//     completion while blockStartRaid is zero, and the clock restarts when
//     the defending Command Ship next goes down.
func MigrateBlockStartRaid(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateBlockStartRaid")

	k := keepers.StructsKeeper
	upgradeHeight := uint64(sdkCtx.BlockHeight())

	var (
		planetsCleared    int
		raidsReanchored   int
	)

	for _, planet := range k.GetAllPlanet(ctx) {
		raidAttributeId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_blockStartRaid, planet.Id)

		if planet.LocationListStart != "" && isDefenderCommandStructVulnerable(ctx, keepers, planet.Owner) {
			k.SetPlanetAttribute(ctx, raidAttributeId, upgradeHeight)
			raidsReanchored++
			logger.Info("re-anchored in-progress raid vulnerability window", "planetId", planet.Id, "raidFleetId", planet.LocationListStart, "blockStartRaid", upgradeHeight)
			continue
		}

		k.SetPlanetAttribute(ctx, raidAttributeId, 0)
		planetsCleared++
	}

	logger.Info("v0.18.0 blockStartRaid normalization complete", "planetsCleared", planetsCleared, "raidsReanchored", raidsReanchored)
	return nil
}

// isDefenderCommandStructVulnerable mirrors
// PlanetCache.IsDefenderCommandStructVulnerable using raw keeper reads
// (no caches, no auto-creation side effects): the defending player's
// Command Ship is offline, destroyed, or non-existent.
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
