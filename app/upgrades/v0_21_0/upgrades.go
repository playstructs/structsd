package v0_21_0

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

// CreateUpgradeHandler returns the v0.21.0 upgrade handler.
//
// Beyond the binary guild-bank conversion changes described in constants.go,
// this upgrade backfills the new guild bank fee fields via MigrateGuildBankFees.
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

		if err := MigrateGuildBankFees(ctx, keepers); err != nil {
			return newVM, err
		}

		// Order matters: rewrite struct types first so the new canDefend flag is
		// populated in state, then prune defender relationships that the flag
		// now invalidates.
		if err := MigrateStructTypes(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateDefenderCanDefend(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateOreClocksToPlanet(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateRaiderArrived(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateJailedReactorEnergy(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigratePrimaryAddressPermissions(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateFleetQueueLimit(ctx, keepers); err != nil {
			return newVM, err
		}

		return newVM, nil
	}
}

// MigrateGuildBankFees backfills the bankConvertInFee / bankConvertOutFee fields
// added to Guild in v0.21.0. Guilds written by an earlier binary have no value
// for these non-nullable LegacyDec fields, so they decode as nil; any arithmetic
// on (or re-marshal of) a nil LegacyDec panics. This sets both to zero for every
// stored guild.
//
// Idempotent: it is a pure backfill of the zero value. A guild that already has
// non-nil fees is left untouched, so a re-run (e.g. replaying the upgrade block)
// produces identical rows.
func MigrateGuildBankFees(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateGuildBankFees")

	k := keepers.StructsKeeper

	var guildsMigrated int
	for _, guild := range k.GetAllGuild(ctx) {
		if guild.NormalizeBankFees() {
			k.SetGuild(ctx, guild)
			guildsMigrated++
		}
	}

	logger.Info("v0.21.0 guild bank fee backfill complete", "guildsMigrated", guildsMigrated)
	return nil
}

// MigrateStructTypes rewrites every struct type from CreateStructTypeGenesis()
// so the new canDefend flag (true for fleet types, false for planetary types)
// is written into state. Without this, struct types persisted by an earlier
// binary would decode canDefend as false for every type, leaving fleet structs
// unable to register as defenders after the upgrade.
//
// Idempotent: it always writes the same canonical genesis values.
func MigrateStructTypes(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateStructTypes")

	k := keepers.StructsKeeper

	structTypes := structstypes.CreateStructTypeGenesis()
	for _, structType := range structTypes {
		k.SetStructType(ctx, structType)
	}

	logger.Info("v0.21.0 struct-type rewrite complete", "structTypesWritten", len(structTypes))
	return nil
}

// MigrateDefenderCanDefend removes every defender registration whose defending
// struct is of a type that can no longer defend (planetary structs after the
// canDefend rule). It uses raw keeper methods (no CurrentContext/StructCache)
// and clears via ClearStructDefender so the same events a StructDefenseClear
// transaction emits (EventStructDefenderClear plus the protectedStructIndex
// EventStructAttribute) are produced, keeping the off-chain indexer in sync.
//
// Cost is O(D) where D is the total number of defender relationships (bounded
// by the number of structs, since each struct defends at most one target).
//
// Idempotent: a re-run finds no remaining invalid registrations.
func MigrateDefenderCanDefend(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateDefenderCanDefend")

	k := keepers.StructsKeeper

	// Struct types are few; memoize to avoid a store read per defender record.
	structTypeCache := make(map[uint64]structstypes.StructType)

	var defendersCleared int
	for _, defender := range k.GetAllStructDefenderExport(ctx) {
		structure, found := k.GetStruct(ctx, defender.DefendingStructId)
		if !found {
			// Orphaned registration; clear it so the indexer drops it too.
			k.ClearStructDefender(ctx, defender.ProtectedStructId, defender.DefendingStructId)
			defendersCleared++
			continue
		}

		structType, cached := structTypeCache[structure.Type]
		if !cached {
			loadedType, typeFound := k.GetStructType(ctx, structure.Type)
			if !typeFound {
				logger.Warn("defender references unknown struct type; skipping", "defendingStructId", defender.DefendingStructId, "structType", structure.Type)
				continue
			}
			structType = loadedType
			structTypeCache[structure.Type] = structType
		}

		if !structType.CanDefend {
			k.ClearStructDefender(ctx, defender.ProtectedStructId, defender.DefendingStructId)
			defendersCleared++
		}
	}

	logger.Info("v0.21.0 defender canDefend prune complete", "defendersCleared", defendersCleared)
	return nil
}

// planetOreMigration holds the per-planet aggregation built while scanning
// structs during MigrateOreClocksToPlanet.
type planetOreMigration struct {
	mineCount   uint64
	refineCount uint64
	mineClock   uint64 // minimum non-zero struct mine clock; 0 means unset
	refineClock uint64 // minimum non-zero struct refine clock; 0 means unset
}

// MigrateOreClocksToPlanet moves ore mine/refine clocks from per-struct
// attributes onto the planet, and seeds oreMiningActiveQuantity /
// oreRefiningActiveQuantity from the count of online planet-located
// mining/refining structs. The planet clock is seeded from the minimum
// existing struct clock so no player loses accrued difficulty age.
//
// Counters are assigned (not incremented) so a re-run is idempotent.
// Old struct-level clock attributes are cleared afterward, emitting
// EventStructAttribute with value 0 so the indexer drops them.
//
// Cost is O(S + P) where S is the number of structs and P the number of
// planets that had at least one mining/refining system online.
func MigrateOreClocksToPlanet(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateOreClocksToPlanet")

	k := keepers.StructsKeeper

	structTypeCache := make(map[uint64]structstypes.StructType)
	byPlanet := make(map[string]*planetOreMigration)
	var structClocksCleared int

	for _, structure := range k.GetAllStruct(ctx) {
		if structure.LocationType != structstypes.ObjectType_planet {
			continue
		}

		structType, cached := structTypeCache[structure.Type]
		if !cached {
			loadedType, typeFound := k.GetStructType(ctx, structure.Type)
			if !typeFound {
				logger.Warn("struct references unknown type; skipping ore clock migration", "structId", structure.Id, "structType", structure.Type)
				continue
			}
			structType = loadedType
			structTypeCache[structure.Type] = structType
		}

		hasMining := structType.HasOreMiningSystem()
		hasRefining := structType.HasOreRefiningSystem()
		if !hasMining && !hasRefining {
			continue
		}

		// Retire the struct-level clocks regardless of online state so the
		// indexer converges on the planet as the only source of truth.
		var mineClock, refineClock uint64
		if hasMining {
			mineAttrId := structskeeper.GetStructAttributeIDByObjectId(structstypes.StructAttributeType_blockStartOreMine, structure.Id)
			if mineClock = k.GetStructAttribute(ctx, mineAttrId); mineClock != 0 {
				k.ClearStructAttribute(ctx, mineAttrId)
				structClocksCleared++
			}
		}
		if hasRefining {
			refineAttrId := structskeeper.GetStructAttributeIDByObjectId(structstypes.StructAttributeType_blockStartOreRefine, structure.Id)
			if refineClock = k.GetStructAttribute(ctx, refineAttrId); refineClock != 0 {
				k.ClearStructAttribute(ctx, refineAttrId)
				structClocksCleared++
			}
		}

		// Only online rigs contribute to the planet counters and clocks.
		statusAttrId := structskeeper.GetStructAttributeIDByObjectId(structstypes.StructAttributeType_status, structure.Id)
		status := structstypes.StructState(k.GetStructAttribute(ctx, statusAttrId))
		if status&structstypes.StructStateOnline == 0 {
			continue
		}

		agg, ok := byPlanet[structure.LocationId]
		if !ok {
			agg = &planetOreMigration{}
			byPlanet[structure.LocationId] = agg
		}

		if hasMining {
			agg.mineCount++
			if mineClock != 0 && (agg.mineClock == 0 || mineClock < agg.mineClock) {
				agg.mineClock = mineClock
			}
		}
		if hasRefining {
			agg.refineCount++
			if refineClock != 0 && (agg.refineClock == 0 || refineClock < agg.refineClock) {
				agg.refineClock = refineClock
			}
		}
	}

	upgradeHeight := uint64(sdkCtx.BlockHeight())
	var planetsUpdated int
	for planetId, agg := range byPlanet {
		if agg.mineCount > 0 {
			mineQtyId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_oreMiningActiveQuantity, planetId)
			k.SetPlanetAttribute(ctx, mineQtyId, agg.mineCount)

			mineClockId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_planetBlockStartOreMine, planetId)
			clock := agg.mineClock
			if existing := k.GetPlanetAttribute(ctx, mineClockId); existing != 0 {
				if clock == 0 || existing < clock {
					clock = existing
				}
			}
			if clock == 0 {
				clock = upgradeHeight
			}
			k.SetPlanetAttribute(ctx, mineClockId, clock)
		}
		if agg.refineCount > 0 {
			refineQtyId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_oreRefiningActiveQuantity, planetId)
			k.SetPlanetAttribute(ctx, refineQtyId, agg.refineCount)

			refineClockId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_planetBlockStartOreRefine, planetId)
			clock := agg.refineClock
			if existing := k.GetPlanetAttribute(ctx, refineClockId); existing != 0 {
				if clock == 0 || existing < clock {
					clock = existing
				}
			}
			if clock == 0 {
				clock = upgradeHeight
			}
			k.SetPlanetAttribute(ctx, refineClockId, clock)
		}
		planetsUpdated++
	}

	logger.Info("v0.21.0 ore clock migration complete",
		"planetsUpdated", planetsUpdated,
		"structClocksCleared", structClocksCleared)
	return nil
}

// MigrateRaiderArrived seeds blockRaiderArrived on every planet that
// currently has a raid in progress (LocationListStart != ""). Without this,
// an in-flight raid would treat the pause window as starting at block 0 and
// shift ore clocks by their entire age at raid end.
//
// The marker is set to the upgrade block height. Idempotent: a re-run
// overwrites with the same height while the raid is still active.
func MigrateRaiderArrived(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateRaiderArrived")

	k := keepers.StructsKeeper
	upgradeHeight := uint64(sdkCtx.BlockHeight())

	var planetsMarked int
	for _, planet := range k.GetAllPlanet(ctx) {
		if planet.LocationListStart == "" {
			continue
		}
		attrId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_blockRaiderArrived, planet.Id)
		k.SetPlanetAttribute(ctx, attrId, upgradeHeight)
		planetsMarked++
	}

	logger.Info("v0.21.0 raider-arrived seed complete", "planetsMarked", planetsMarked)
	return nil
}

// MigrateJailedReactorEnergy zeroes the energy ratio on every reactor whose
// validator is currently jailed or no longer exists in staking.
//
// From this release forward, reactor energy is gated by validator health and the
// AfterValidatorBeginUnbonding hook applies that gate as jails happen. Reactors
// jailed before the upgrade never passed through that hook, so without this
// backfill they would keep producing energy indefinitely, which is the exact
// exploit the release closes.
//
// Missing validators are gated as well, matching the fail-closed behaviour of
// reactorEnergyRatio: a reactor whose validator has been removed from staking
// entirely has no claim to energy.
//
// Idempotent. ReactorGateEnergy skips infusions already at a zero ratio, so a
// re-run writes nothing and emits no duplicate indexer events. Recovery for any
// reactor gated here is the same as for any other: it unjails and rebonds, or
// someone sends MsgReactorRestart.
func MigrateJailedReactorEnergy(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateJailedReactorEnergy")

	k := keepers.StructsKeeper

	var reactorsGated int
	for _, reactor := range k.GetAllReactor(ctx) {
		valAddr, err := sdk.ValAddressFromBech32(reactor.Validator)
		if err != nil {
			logger.Error("skipping reactor with unparsable validator address",
				"reactorId", reactor.Id, "validator", reactor.Validator, "error", err)
			continue
		}

		// Read through the structs keeper's staking adapter rather than
		// keepers.StakingKeeper so this migration is exercisable with the same
		// mock staking keeper the structs unit tests use, matching v0.17.0.
		validator, validatorErr := k.StakingKeeper().GetValidator(ctx, valAddr)
		if validatorErr == nil && !validator.IsJailed() {
			continue
		}

		k.ReactorGateEnergy(ctx, valAddr)
		reactorsGated++
	}

	logger.Info("v0.21.0 jailed reactor energy gate complete", "reactorsGated", reactorsGated)
	return nil
}

// MigratePrimaryAddressPermissions grants PermAll to every player's current
// primary address.
//
// PlayerUpdatePrimaryAddress now requires the caller to hold PermAll, matching
// the unconditional grant SetPrimaryAddress writes onto the incoming address. A
// player can reduce their own primary address below PermAll via
// permission-set/revoke-on-address and cannot restore bits to themself, so
// without this backfill those accounts would be permanently locked out of the
// primary-address swap.
//
// Idempotent: addresses that already hold PermAll are left untouched, so a
// re-run writes nothing and emits no duplicate EventPermission.
func MigratePrimaryAddressPermissions(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migratePrimaryAddressPermissions")

	k := keepers.StructsKeeper

	var addressesUpgraded int
	for _, player := range k.GetAllPlayer(ctx) {
		if player.PrimaryAddress == "" {
			continue
		}

		permissionId := structskeeper.GetAddressPermissionIDBytes(player.PrimaryAddress)
		current := k.GetPermissionsByBytes(ctx, permissionId)
		if current == structstypes.PermAll {
			continue
		}

		k.SetPermissionsByBytes(ctx, permissionId, structstypes.PermAll)
		addressesUpgraded++
	}

	logger.Info("v0.21.0 primary address permission normalize complete",
		"addressesUpgraded", addressesUpgraded)
	return nil
}

// MigrateFleetQueueLimit enforces the new per-planet raid-queue capacity
// (1 + locationListExtra, default extra=0 => capacity 1).
//
// For each planet it:
//  1. Walks locationListStart -> Backward* to discover the live visitor order.
//  2. Seeds locationListCount to that length so subsequent SetLocationToPlanet
//     decrements stay consistent (the field did not exist before this release).
//  3. Leaves locationListExtra at 0.
//  4. Sends every fleet after the head home via SetLocationToPlanet.
//
// Idempotent under capacity 1: a re-run finds at most one visitor, seeds
// count=1 (or 0), and has no overflow to evict.
func MigrateFleetQueueLimit(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateFleetQueueLimit")

	k := keepers.StructsKeeper

	var planetsTouched int
	var fleetsSentHome int

	for _, planet := range k.GetAllPlanet(ctx) {
		fleetIds := make([]string, 0)
		for fleetId := planet.LocationListStart; fleetId != ""; {
			fleet, found := k.GetFleet(ctx, fleetId)
			if !found {
				logger.Error("raid queue points at missing fleet; stopping walk",
					"planetId", planet.Id, "fleetId", fleetId)
				break
			}
			fleetIds = append(fleetIds, fleetId)
			fleetId = fleet.LocationListBackward
		}

		// Seed the denormalized counter from the live list before any eviction
		// so SetLocationToPlanet decrements from a correct baseline.
		if planet.LocationListCount != uint64(len(fleetIds)) {
			planet.LocationListCount = uint64(len(fleetIds))
			k.SetPlanet(ctx, planet)
		}

		capacity := uint64(1) + planet.LocationListExtra
		if uint64(len(fleetIds)) <= capacity {
			if len(fleetIds) > 0 {
				planetsTouched++
			}
			continue
		}

		cc := k.NewCurrentContext(ctx)
		// Evict tail-first so middle-unlink is rare and head stays put.
		for i := len(fleetIds) - 1; i >= int(capacity); i-- {
			fleet, err := cc.GetFleetById(fleetIds[i])
			if err != nil {
				logger.Error("failed to load overflow fleet",
					"planetId", planet.Id, "fleetId", fleetIds[i], "error", err)
				continue
			}
			home := fleet.GetOwner().GetPlanet()
			if home == nil || !home.LoadPlanet() {
				logger.Error("overflow fleet has no home planet; skipping",
					"planetId", planet.Id, "fleetId", fleetIds[i], "owner", fleet.GetOwnerId())
				continue
			}
			fleet.SetLocationToPlanet(home)
			fleetsSentHome++
		}
		cc.CommitAll()
		planetsTouched++
	}

	logger.Info("v0.21.0 fleet queue limit migration complete",
		"planetsTouched", planetsTouched,
		"fleetsSentHome", fleetsSentHome)
	return nil
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
