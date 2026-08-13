package v0_21_0

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

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

		// The binary's bank send restriction is already active while this handler
		// runs. Build the provider-pool allowlist before any migration below moves
		// guild-denom balances through those pools.
		MigrateProviderPoolAddressIndex(ctx, keepers)
		MigrateProtectLegacyGuildEscrow(ctx, keepers)

		if err := MigrateGuildBankFees(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateGuildNameIndex(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateGuildJoinBypassLevels(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateGuildCharter(ctx, keepers); err != nil {
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

		// Before the reconcile below, which recomputes each infusion's capacity
		// contribution and so needs the owner it is crediting to be correct.
		if err := MigrateInfusionOwnership(ctx, keepers); err != nil {
			return newVM, err
		}

		// Both read staking rather than structs state, so they are independent
		// of the infusion work around them. The repair runs before the audit so
		// that the audit's log describes the state the chain is left in.
		if err := MigrateDelegationDistributionState(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateAuditDelegatorShares(ctx, keepers); err != nil {
			return newVM, err
		}

		// After MigrateJailedReactorEnergy, which gates whole reactors. This
		// then rebuilds each infusion from live staking state, so the two agree
		// on a jailed validator's zero ratio and this has the final say on the
		// fuel behind it.
		if err := MigrateReconcileReactorInfusions(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigratePrimaryAddressPermissions(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateFleetQueueLimit(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateAgreementCheckpointOverbill(ctx, keepers); err != nil {
			return newVM, err
		}

		// After MigrateAgreementCheckpointOverbill, so that each settlement is paid
		// out of a collateral pool that has already had its over-billed revenue
		// returned to it.
		if err := MigrateExpireOverdueAgreements(ctx, keepers); err != nil {
			return newVM, err
		}

		// After MigrateOreClocksToPlanet, which seeds the ore counters this then
		// recomputes: both derive the same number from the same online structs, and
		// running the repair last means it has the final say.
		if err := MigrateStructPhantomAggregates(ctx, keepers); err != nil {
			return newVM, err
		}

		// After everything that can destroy an allocation, so this does not hand a
		// new controller an allocation that is about to be torn down anyway.
		if err := MigrateOrphanedAllocationControllers(ctx, keepers); err != nil {
			return newVM, err
		}

		// Last, so that the rebuilt index reflects the allocations that actually
		// survive this upgrade. Anything above that settles an agreement or sheds
		// load can destroy allocations, and the rebuild has to have the final say.
		if err := MigrateAutoResizeAllocationIndex(ctx, keepers); err != nil {
			return newVM, err
		}

		return newVM, nil
	}
}

// MigrateProviderPoolAddressIndex rebuilds the reverse lookup required by the
// guild-token send restriction and confiscation policy. It must run before any
// migration that sends a guild denom into or between provider pools.
func MigrateProviderPoolAddressIndex(ctx context.Context, keepers *upgrades.Keepers) {
	for _, provider := range keepers.StructsKeeper.GetAllProvider(ctx) {
		keepers.StructsKeeper.IndexProviderPoolAddresses(ctx, provider.Id)
	}
}

// MigrateProtectLegacyGuildEscrow records v1 transfer escrows that already hold
// native guild tokens. New escrow is rejected by the bank send restriction, but
// balances written by an older binary still back vouchers and must not be
// confiscated out from under them.
func MigrateProtectLegacyGuildEscrow(ctx context.Context, keepers *upgrades.Keepers) {
	keepers.StructsKeeper.ProtectLegacyGuildEscrowBalances(ctx)
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

/* MigrateGuildCharter makes the new proof-of-work guild charter usable.
 *
 * Three writes, and each of them is load-bearing rather than cosmetic:
 *
 * The params. A Params record written by an earlier binary decodes the two new
 * fields as zero, and zero is the one difficulty range CalculateDifficulty
 * cannot take — it would pin the requirement at 64 leading zeros forever, which
 * is guild creation being permanently impossible. Only unset fields are filled,
 * so a chain that has already tuned them keeps its values.
 *
 * The anchor. It is the height the difficulty decays from and cannot be derived
 * from anything else on disk. Left unset, the first read would fall back to the
 * current height anyway, but stamping it explicitly is what makes the first
 * proof-founded guild land a predictable window after the upgrade rather than
 * depending on when someone first asks.
 *
 * The reactors. Eligibility is a stored height, so an existing reactor has none
 * and would never become eligible. They are stamped at the upgrade height, which
 * makes them all eligible at once: the accepted launch burst is one free guild
 * for every currently bonded validator whose reactor has not already founded
 * one. A reactor whose GuildId is already set is spent and cannot double-dip.
 *
 * Idempotent. Params and reactors are only written where the field is unset, and
 * a re-run at the same height stamps the same anchor.
 */
func MigrateGuildCharter(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateGuildCharter")

	k := keepers.StructsKeeper
	upgradeHeight := uint64(sdkCtx.BlockHeight())

	params := k.GetParams(ctx)
	if params.GuildCharterDifficultyRange < structstypes.MinGuildCharterDifficultyRange {
		params.GuildCharterDifficultyRange = structstypes.DefaultGuildCharterDifficultyRange
	}
	if params.GuildCharterReactorAge == 0 {
		params.GuildCharterReactorAge = structstypes.DefaultGuildCharterReactorAge
	}
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	k.SetGuildCharterAnchor(ctx, upgradeHeight)

	var reactorsStamped, alreadySpent int
	for _, reactor := range k.GetAllReactor(ctx) {
		if reactor.GuildCharterEligibleHeight != 0 {
			continue
		}
		if reactor.GuildId != "" {
			alreadySpent++
		}

		reactor.GuildCharterEligibleHeight = upgradeHeight
		k.SetReactor(ctx, reactor)
		reactorsStamped++
	}

	logger.Info("v0.21.0 guild charter migration complete",
		"anchor", upgradeHeight,
		"difficultyRange", params.GuildCharterDifficultyRange,
		"reactorAge", params.GuildCharterReactorAge,
		"reactorsStamped", reactorsStamped,
		"reactorsAlreadySpent", alreadySpent)
	return nil
}

// MigrateGuildJoinBypassLevels clamps guild join bypass levels that hold a value
// outside the three declared in the guildJoinBypassLevel enum.
//
// Before v0.21.0 the update handlers wrote msg.GuildJoinBypassLevel straight
// through, and proto3 enums are open, so any int32 both decoded and persisted.
// The readers in guild_cache.go now deny an undeclared level instead of falling
// through their switches, which is what closes the membership bypass; this
// migration is about the records themselves. Left alone they would be inert but
// malformed: permanently closed to joins with no signal as to why, and enough to
// fail GenesisState.Validate on the next export.
//
// Closed is the safe direction and a recoverable one. CanUpdateJoinConstraintsBy
// reads only PermGuildJoinConstraintsUpdate, never the bypass level, so a guild
// this touches is reopened by its owner in one transaction.
//
// Idempotent: it rewrites only guilds NormalizeJoinBypassLevels reports as
// changed, and closed is a declared value, so a second run finds nothing.
func MigrateGuildJoinBypassLevels(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateGuildJoinBypassLevels")

	k := keepers.StructsKeeper

	var guildsMigrated int
	for _, guild := range k.GetAllGuild(ctx) {
		byRequest := guild.JoinInfusionMinimumBypassByRequest
		byInvite := guild.JoinInfusionMinimumBypassByInvite

		if guild.NormalizeJoinBypassLevels() {
			// Error level: no correct binary writes an undeclared level, so a
			// hit here means the guild was exposed to the membership bypass and
			// its owner needs to know their join policy was reset.
			logger.Error("guild holds an undeclared join bypass level; clamping to closed",
				"guildId", guild.Id, "byRequest", int32(byRequest), "byInvite", int32(byInvite))
			k.SetGuild(ctx, guild)
			guildsMigrated++
		}
	}

	logger.Info("v0.21.0 guild join bypass level normalization complete", "guildsMigrated", guildsMigrated)
	return nil
}

// MigrateGuildNameIndex rebuilds the guild name index under the pinned Unicode
// normalization, and is expected to write back exactly the rows it found.
//
// v0.21.0 stops reading the compiling toolchain's Unicode tables. NormalizeName
// now case-folds and trims through checked-in Unicode 15.0.0 data, and its
// output is a KV key: SetGuildNameIndex stores "Guild/name/" + NormalizeName.
// Every Go release the chain has run so far ships Unicode 15.0.0, so the new
// normalization produces the same bytes as the old one -- proven exhaustively
// over the whole rune space by TestNormalizeNameMatchesLegacyForm and
// TestPinnedTablesMatchToolchain -- which makes this a no-op on any state a
// correct binary produced.
//
// It runs anyway because it is cheap and because it is the only thing that would
// repair state a divergent binary could have written, and because the guild name
// index cannot be reconstructed later: RemoveGuildNameIndex deletes by
// re-normalizing a name, so a row whose key the current normalization no longer
// produces is unreachable and would sit there forever holding a name hostage.
//
// A name it cannot carry forward is dropped rather than preserved. That costs
// the guild one transaction to set the name again, and the alternative -- an
// index row that does not match the guild's stored name -- lets a later rename
// delete some other guild's row. Both drop paths log at error level because
// neither is reachable from state a correct binary wrote.
//
// Idempotent: a second run clears the rows it just wrote and rebuilds the same
// ones from the same guilds, and a name it already dropped is empty and skipped.
func MigrateGuildNameIndex(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateGuildNameIndex")

	k := keepers.StructsKeeper

	before := k.GetAllGuildNameIndex(ctx)
	cleared := k.ClearGuildNameIndex(ctx)

	var (
		indexed  int
		skipped  int
		dropped  int
		rekeyed  int
		collided int
	)

	// GetAllGuild walks the guild prefix in key order, so which guild wins a
	// collision is the same on every node.
	claimedBy := make(map[string]string)
	for _, guild := range k.GetAllGuild(ctx) {
		if guild.Name == "" {
			skipped++
			continue
		}

		// Re-validate rather than trust the stored value. A name that fails the
		// pinned rules is one no correct binary could have written, and leaving
		// it indexed would key state to a name the chain would now refuse.
		if err := structstypes.ValidateEntityName(guild.Name); err != nil {
			logger.Error("guild name is invalid under the pinned Unicode rules; clearing it",
				"guildId", guild.Id, "name", guild.Name, "err", err)
			guild.Name = ""
			k.SetGuild(ctx, guild)
			dropped++
			continue
		}

		key := structstypes.NormalizeName(guild.Name)
		if owner, taken := claimedBy[key]; taken {
			logger.Error("two guilds normalize to the same name index key; clearing the later one",
				"guildId", guild.Id, "name", guild.Name, "key", key, "keptBy", owner)
			guild.Name = ""
			k.SetGuild(ctx, guild)
			collided++
			dropped++
			continue
		}

		claimedBy[key] = guild.Id
		k.SetGuildNameIndex(ctx, guild.Name, guild.Id)
		indexed++

		if previous, existed := before[key]; !existed || previous != guild.Id {
			rekeyed++
		}
	}

	// Loud when it is not the no-op it should be. Any non-zero count here means
	// a binary wrote guild state under different Unicode tables than these, and
	// that is the condition the whole change exists to make impossible.
	if rekeyed > 0 || dropped > 0 || cleared != indexed {
		logger.Error("v0.21.0 guild name index rebuild was not a no-op",
			"rowsBefore", cleared, "rowsAfter", indexed,
			"rowsMovedOrAdded", rekeyed, "namesDropped", dropped, "collisions", collided)
	}

	logger.Info("v0.21.0 guild name index rebuild complete",
		"rowsBefore", cleared, "rowsAfter", indexed,
		"guildsWithoutName", skipped, "namesDropped", dropped, "collisions", collided)
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

// MigrateReconcileReactorInfusions rebuilds every reactor infusion from live
// Cosmos staking state, clearing the phantom fuel left behind by full
// redelegations.
//
// Until this release Hooks.BeforeDelegationRemoved was a no-op. Staking's Unbond
// routes a delegation whose shares reach zero through RemoveDelegation, which
// fires only that hook and skips AfterDelegationModified, so a full redelegation
// left the source infusion's Fuel, Power and grid contributions installed while
// the destination was granted capacity for the same stake. GuildMembershipJoin
// redelegates a player's entire infusion, so this is not confined to deliberate
// abuse: ordinary players accrued phantom capacity every time they joined a
// guild on a different reactor.
//
// Capacity that no stake backs has to go, and removing it is deliberately
// allowed to have consequences. Each cleared row queues a grid cascade, and the
// EndBlocker of the upgrade block sheds allocations in creation order until load
// fits capacity again, which can cascade downstream through substations and can
// tear down provider agreements. That is the correct outcome for capacity that
// was never real, and the counters logged here are the record of how much of it
// there was.
//
// Idempotent. reconcileInfusionForDelegation guards each write on the value
// changing, so a re-run writes nothing and emits no duplicate EventInfusion.
func MigrateReconcileReactorInfusions(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateReconcileReactorInfusions")

	k := keepers.StructsKeeper

	var (
		infusionsVisited     int
		infusionsChanged     int
		phantomFuelCleared   uint64
		phantomPowerCleared  uint64
		commissionDivergence int
		playersAffected      = make(map[string]struct{})
	)

	for _, reactor := range k.GetAllReactor(ctx) {
		valAddr, err := sdk.ValAddressFromBech32(reactor.Validator)
		if err != nil {
			logger.Error("skipping reactor with unparsable validator address",
				"reactorId", reactor.Id, "validator", reactor.Validator, "error", err)
			continue
		}

		for _, before := range k.GetAllInfusionsByDestination(ctx, reactor.Id) {
			playerAddress, addrErr := sdk.AccAddressFromBech32(before.Address)
			if addrErr != nil {
				logger.Error("skipping infusion with unparsable delegator address",
					"reactorId", reactor.Id, "address", before.Address, "error", addrErr)
				continue
			}

			infusionsVisited++

			// Commission is only ever written from the reactor default, so a
			// divergence means the record predates that or arrived through a
			// genesis import. The reconcile re-bases it; count it so the
			// re-basing is visible rather than silent.
			if before.Commission.IsNil() || reactor.DefaultCommission.IsNil() ||
				!before.Commission.Equal(reactor.DefaultCommission) {
				commissionDivergence++
			}

			k.ReconcileInfusionForDelegation(ctx, playerAddress, valAddr)

			after, found := k.GetInfusion(ctx, reactor.Id, before.Address)
			if !found {
				logger.Error("infusion vanished during reconcile",
					"reactorId", reactor.Id, "address", before.Address)
				continue
			}

			if after.Fuel == before.Fuel && after.Power == before.Power {
				continue
			}

			infusionsChanged++
			playersAffected[before.PlayerId] = struct{}{}

			if after.Fuel < before.Fuel {
				phantomFuelCleared += before.Fuel - after.Fuel
			}
			if after.Power < before.Power {
				phantomPowerCleared += before.Power - after.Power
			}

			logger.Info("reconciled reactor infusion",
				"reactorId", reactor.Id, "address", before.Address, "playerId", before.PlayerId,
				"fuelBefore", before.Fuel, "fuelAfter", after.Fuel,
				"powerBefore", before.Power, "powerAfter", after.Power)
		}
	}

	logger.Info("v0.21.0 reactor infusion reconcile complete",
		"infusionsVisited", infusionsVisited,
		"infusionsChanged", infusionsChanged,
		"phantomFuelCleared", phantomFuelCleared,
		"phantomPowerCleared", phantomPowerCleared,
		"playersAffected", len(playersAffected),
		"commissionDivergence", commissionDivergence)

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

// MigrateAgreementCheckpointOverbill returns the one block of revenue every
// existing agreement was over-billed for.
//
// Agreements opened before v0.21.0 start serving at openHeight+1, but
// AgreementOpen raises the provider's load immediately and checkpoints the
// provider at openHeight. Checkpoint bills aggregate load from the checkpoint
// block, so the first checkpoint span covering an agreement charged one block of
// service its consumer never received. v0.21.0 aligns the two by starting service
// in the block the agreement opens, which fixes new agreements but leaves every
// stored one with a collateral pool short by
// capacity * rate * (1 - providerCancellationPenalty), which the
// provider-collateral-solvency invariant reports as insolvency.
//
// The over-billed amount went to the provider's earnings pool, so that is where
// it comes back from. Nothing is taken from the consumer: shifting the agreement
// window instead would either widen the consumer's penalty claim (making the
// deficit worse) or silently cut a block of service they paid for.
//
// The claw-back is clamped to what the earnings pool still holds, matching
// ProviderCache.SweepRevenue. A provider who has already withdrawn cannot be
// pursued, and any residual shortfall is logged and left to settle the way it
// would anyway: consumer payouts are exact and take priority, so the provider
// absorbs it through the clamp on their own revenue.
//
// Not idempotent. A stored agreement carries no marker distinguishing a realigned
// window from an original one, so a second run would move another block's worth.
// It is bounded (one block per agreement, clamped to the earnings pool) and only
// ever moves value from provider to consumer, so a replay is safe if not exact.
func MigrateAgreementCheckpointOverbill(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateAgreementCheckpointOverbill")

	k := keepers.StructsKeeper

	var providersReconciled int
	var providersShort int
	totalReturned := math.ZeroInt()

	for _, provider := range k.GetAllProvider(ctx) {
		// Same guard as the invariant: nothing is owed against a provider with no
		// usable published rate, and the arithmetic would be meaningless.
		if provider.Rate.Denom == "" || provider.Rate.Amount.IsNil() || provider.ProviderCancellationPenalty.IsNil() {
			continue
		}

		// One block per agreement at that agreement's own capacity, which is what
		// the aggregate-load checkpoint actually over-charged.
		overbill := math.ZeroInt()
		for _, agreement := range k.GetAllAgreementByProviderIndex(ctx, provider.Id) {
			blockValue := math.LegacyNewDecFromInt(provider.Rate.Amount.Mul(math.NewIntFromUint64(agreement.Capacity)))
			overbill = overbill.Add(blockValue.Sub(blockValue.Mul(provider.ProviderCancellationPenalty)).TruncateInt())
		}

		if !overbill.IsPositive() {
			continue
		}

		earningsPool := structskeeper.GetProviderEarningsPoolLocation(provider.Id)
		available := k.BankKeeper().SpendableCoin(ctx, earningsPool, provider.Rate.Denom).Amount

		returned := overbill
		if available.LT(returned) {
			returned = available
			providersShort++
			logger.Error("provider earnings pool cannot cover the checkpoint overbill; leaving the remainder to the revenue clamp",
				"providerId", provider.Id,
				"overbill", overbill.String(),
				"available", available.String())
		}

		if !returned.IsPositive() {
			continue
		}

		collateralPool := structskeeper.GetProviderCollateralPoolLocation(provider.Id)
		coins := sdk.NewCoins(sdk.NewCoin(provider.Rate.Denom, returned))
		if err := k.BankKeeper().SendCoins(ctx, earningsPool, collateralPool, coins); err != nil {
			// A transfer failure here is not worth halting the upgrade over: the
			// pool stays short and the revenue clamp absorbs it, exactly as it
			// would have without this migration.
			logger.Error("failed to return checkpoint overbill to the collateral pool",
				"providerId", provider.Id, "amount", returned.String(), "error", err)
			continue
		}

		totalReturned = totalReturned.Add(returned)
		providersReconciled++
	}

	logger.Info("v0.21.0 agreement checkpoint overbill reconciliation complete",
		"providersReconciled", providersReconciled,
		"providersShort", providersShort,
		"totalReturned", totalReturned.String())
	return nil
}

// MigrateExpireOverdueAgreements settles every agreement whose end block has
// already passed.
//
// Expiry is driven from the EndBlocker by AgreementExpirations, which reads the
// expiration index at exactly the current height. There is no range scan and no
// retry, so an agreement not torn down on the one block it comes up is never
// revisited: it keeps its capacity in the provider's load and Checkpoint goes on
// billing that capacity against the shared collateral pool every block, funding
// the overcharge out of other consumers' collateral.
//
// v0.21.0 makes that unreachable by stopping Expire from abandoning the teardown
// when the allocation it meant to destroy has already gone. This clears anything
// that reached the state before the fix, so the new agreement-expiry-liveness
// invariant starts clean. On a healthy chain it settles nothing.
func MigrateExpireOverdueAgreements(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateExpireOverdueAgreements")

	k := keepers.StructsKeeper
	currentBlock := uint64(sdkCtx.BlockHeight())

	// Collect the ids up front. Settling an agreement removes rows and can reach
	// other agreements through the shared provider, so the list must not be built
	// lazily over a store that is being written underneath it.
	var overdue []string
	for _, agreement := range k.GetAllAgreement(ctx) {
		if agreement.EndBlock < currentBlock {
			overdue = append(overdue, agreement.Id)
		}
	}

	if len(overdue) == 0 {
		logger.Info("v0.21.0 overdue agreement sweep found nothing to settle")
		return nil
	}

	cc := k.NewCurrentContext(ctx)

	var settled, failed int
	for _, agreementId := range overdue {
		agreement := cc.GetAgreement(agreementId)
		if !agreement.LoadAgreement() {
			continue
		}

		if err := agreement.Expire(); err != nil {
			// Same posture as the EndBlocker this stands in for: one that cannot be
			// settled is logged and the rest still go, rather than failing the
			// upgrade over state that is already broken.
			logger.Error("overdue agreement could not be settled",
				"agreementId", agreementId, "error", err)
			failed++
			continue
		}
		settled++
	}

	cc.CommitAll()

	logger.Info("v0.21.0 overdue agreement sweep complete",
		"settled", settled, "failed", failed)
	return nil
}

// planetAggregate is the per-planet recompute built while scanning structs in
// MigrateStructPhantomAggregates.
type planetAggregate struct {
	shield       uint64
	cannons      uint64
	interceptors uint64
	mining       uint64
	refining     uint64
}

// MigrateStructPhantomAggregates rebuilds every aggregate that struct
// destruction and reactivation could corrupt, from the structs that are actually
// still standing.
//
// Two bugs fed it. Destruction was not idempotent and left the struct in its
// planet slot, so a cancelled build that was then caught by planet completion
// released its BuildDraw reservation and its type count twice — understating the
// owner's load and the count of structs they own. And a destroyed struct still
// read as built and offline, so it could be activated during the sweep window;
// GoOnline re-added its planetary shield, defensive cannon or interceptor count
// and ore rig, and the sweep then deleted the struct without taking it offline
// again, leaving those contributions behind with no object to reverse them.
//
// None of that is recoverable by inspecting the damage: the surviving structs are
// the only record of what the totals should be, so this recomputes rather than
// adjusts. Destroyed structs are excluded, since destruction already removed
// their contributions and the sweep is about to remove the struct.
//
// Cost is O(S + P + G): one pass over structs, one over planets, one over the
// grid attribute store to find rows keyed to structs that no longer exist.
//
// Idempotent: every value is derived and assigned, never adjusted, so a re-run
// writes the same numbers. Values that already agree are skipped so no
// redundant indexer event is emitted.
func MigrateStructPhantomAggregates(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateStructPhantomAggregates")

	k := keepers.StructsKeeper

	structTypeCache := make(map[uint64]structstypes.StructType)
	structType := func(structure structstypes.Struct) (structstypes.StructType, bool) {
		if cached, ok := structTypeCache[structure.Type]; ok {
			return cached, true
		}
		loaded, found := k.GetStructType(ctx, structure.Type)
		if !found {
			logger.Warn("struct references unknown type; excluding from the recompute",
				"structId", structure.Id, "structType", structure.Type)
			return structstypes.StructType{}, false
		}
		structTypeCache[structure.Type] = loaded
		return loaded, true
	}

	// Every player carries PlayerPassiveDraw whether or not they own a struct, so
	// seed from the player list rather than from the structs: a player whose only
	// struct was destroyed still needs their load corrected.
	playerLoad := make(map[string]uint64)
	for _, player := range k.GetAllPlayer(ctx) {
		playerLoad[player.Id] = structstypes.PlayerPassiveDraw
	}

	byPlanet := make(map[string]*planetAggregate)
	planetAgg := func(planetId string) *planetAggregate {
		agg, ok := byPlanet[planetId]
		if !ok {
			agg = &planetAggregate{}
			byPlanet[planetId] = agg
		}
		return agg
	}

	// typeCount is keyed by owner and struct type together.
	type ownerType struct {
		owner  string
		typeId uint64
	}
	typeCount := make(map[ownerType]uint64)

	liveStructs := make(map[string]bool)

	for _, structure := range k.GetAllStruct(ctx) {
		liveStructs[structure.Id] = true

		status := structstypes.StructState(k.GetStructAttribute(ctx,
			structskeeper.GetStructAttributeIDByObjectId(structstypes.StructAttributeType_status, structure.Id)))
		if status&structstypes.StructStateDestroyed != 0 {
			continue
		}

		sType, found := structType(structure)
		if !found {
			continue
		}

		typeCount[ownerType{owner: structure.Owner, typeId: structure.Type}]++

		// BuildDraw is reserved while building and swapped for PassiveDraw when the
		// build completes and the struct comes online, which is why these are
		// independent rather than exclusive.
		if _, known := playerLoad[structure.Owner]; known {
			if status&structstypes.StructStateBuilt == 0 {
				playerLoad[structure.Owner] += sType.BuildDraw
			}
			if status&structstypes.StructStateOnline != 0 {
				playerLoad[structure.Owner] += sType.PassiveDraw
			}
		} else {
			logger.Warn("struct owned by an unknown player; excluding from the load recompute",
				"structId", structure.Id, "owner", structure.Owner)
		}

		if status&structstypes.StructStateOnline == 0 {
			continue
		}

		// Planet-keyed contributions follow the struct's location the way
		// StructCache.GetPlanet does: a fleet struct contributes to whatever planet
		// its fleet is currently at.
		var planetId string
		switch structure.LocationType {
		case structstypes.ObjectType_planet:
			planetId = structure.LocationId
		case structstypes.ObjectType_fleet:
			fleet, fleetFound := k.GetFleet(ctx, structure.LocationId)
			if !fleetFound || fleet.LocationType != structstypes.ObjectType_planet {
				continue
			}
			planetId = fleet.LocationId
		default:
			continue
		}
		if planetId == "" {
			continue
		}

		agg := planetAgg(planetId)
		if sType.HasOreReserveDefensesSystem() {
			agg.shield += sType.PlanetaryShieldContribution
		}
		if sType.HasPlanetaryDefensesSystem() {
			switch sType.PlanetaryDefenses {
			case structstypes.TechPlanetaryDefenses_defensiveCannon:
				agg.cannons++
			case structstypes.TechPlanetaryDefenses_lowOrbitBallisticInterceptorNetwork:
				agg.interceptors++
			}
		}
		if sType.HasOreMiningSystem() {
			agg.mining++
		}
		if sType.HasOreRefiningSystem() {
			agg.refining++
		}
	}

	var loadsRepaired int
	for playerId, expected := range playerLoad {
		attrId := structskeeper.GetGridAttributeIDByObjectId(structstypes.GridAttributeType_structsLoad, playerId)
		if k.GetGridAttribute(ctx, attrId) == expected {
			continue
		}
		k.SetGridAttribute(ctx, attrId, expected)
		loadsRepaired++
	}

	var typeCountsRepaired int
	for key, expected := range typeCount {
		attrId := structskeeper.GetStructAttributeIDByObjectIdAndSubIndex(
			structstypes.StructAttributeType_typeCount, key.owner, key.typeId)
		if k.GetStructAttribute(ctx, attrId) == expected {
			continue
		}
		k.SetStructAttribute(ctx, attrId, expected)
		typeCountsRepaired++
	}

	var planetsRepaired int
	for _, planet := range k.GetAllPlanet(ctx) {
		agg, ok := byPlanet[planet.Id]
		if !ok {
			agg = &planetAggregate{}
		}

		shieldAttrId := structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_planetaryShield, planet.Id)
		currentShield := k.GetPlanetAttribute(ctx, shieldAttrId)

		// PlanetaryShieldBase is written when the planet is created and is never
		// cleared afterwards, not even on completion. The one exception is a planet
		// imported at genesis in a non-active status, which never receives it, so
		// take the base from what the planet actually holds rather than assuming it
		// and inventing a shield those planets never had.
		base := uint64(structstypes.PlanetaryShieldBase)
		if planet.Status != structstypes.PlanetStatus_active && currentShield < base {
			base = 0
		}

		repaired := false
		for _, field := range []struct {
			attrId   string
			expected uint64
		}{
			{shieldAttrId, base + agg.shield},
			{structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_defensiveCannonQuantity, planet.Id), agg.cannons},
			{structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity, planet.Id), agg.interceptors},
			{structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_oreMiningActiveQuantity, planet.Id), agg.mining},
			{structskeeper.GetPlanetAttributeIDByObjectId(structstypes.PlanetAttributeType_oreRefiningActiveQuantity, planet.Id), agg.refining},
		} {
			if k.GetPlanetAttribute(ctx, field.attrId) == field.expected {
				continue
			}
			k.SetPlanetAttribute(ctx, field.attrId, field.expected)
			repaired = true
		}
		if repaired {
			planetsRepaired++
		}
	}

	// Grid rows keyed to a struct that no longer exists. DestroyAndCommit clears
	// these for power generators, but a struct reactivated after destruction wrote
	// them again and the sweep deletes the object without a second pass.
	structPrefix := fmt.Sprintf("%d-", structstypes.ObjectType_struct)
	orphanTypes := map[string]bool{}
	for _, attributeType := range []structstypes.GridAttributeType{
		structstypes.GridAttributeType_ready,
		structstypes.GridAttributeType_load,
		structstypes.GridAttributeType_capacity,
		structstypes.GridAttributeType_fuel,
		structstypes.GridAttributeType_power,
	} {
		orphanTypes[fmt.Sprintf("%d-", attributeType)] = true
	}

	var orphansCleared int
	for _, record := range k.GetAllGridExport(ctx) {
		split := strings.SplitN(record.AttributeId, "-", 2)
		if len(split) != 2 {
			continue
		}
		if !orphanTypes[split[0]+"-"] {
			continue
		}
		objectId := split[1]
		if !strings.HasPrefix(objectId, structPrefix) || liveStructs[objectId] {
			continue
		}
		k.ClearGridAttribute(ctx, record.AttributeId)
		orphansCleared++
	}

	logger.Info("v0.21.0 struct phantom aggregate repair complete",
		"loadsRepaired", loadsRepaired,
		"typeCountsRepaired", typeCountsRepaired,
		"planetsRepaired", planetsRepaired,
		"orphanGridAttributesCleared", orphansCleared)
	return nil
}

// MigrateAutoResizeAllocationIndex rebuilds the auto-resize index from the
// allocations that actually exist.
//
// The index maps a source object id to the automated allocation riding on it,
// but AllocationCache.Destroy cleared it with the allocation's own id, so the
// delete matched nothing and every automated allocation ever torn down left its
// hook behind. A leaked hook bricks its source: SetSource refuses any new
// automated allocation on a source the index mentions, without checking that the
// allocation still exists, and the infusion capacity path treats the hook as live
// and so skips the grid cascade that a capacity cut should trigger.
//
// Rebuilding rather than pruning fixes every way the index can be wrong in one
// pass — entries whose allocation is gone, entries naming a non-automated
// allocation, entries filed under a source the allocation does not claim, and any
// automated allocation missing a hook.
//
// Idempotent: a second run clears the index it just wrote and writes the same
// rows again, since it derives everything from the allocations themselves.
func MigrateAutoResizeAllocationIndex(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateAutoResizeAllocationIndex")

	k := keepers.StructsKeeper

	before := make(map[string]string)
	for _, hook := range k.GetAllAutoResizeAllocationSource(ctx) {
		before[hook.SourceObjectId] = hook.AllocationId
	}

	cleared := k.ClearAllAutoResizeAllocationSource(ctx)

	var (
		indexed  int
		added    int
		rekeyed  int
		collided int
	)

	// GetAllAllocation walks the allocation prefix in key order, so which
	// allocation wins a collision is the same on every node.
	claimedBy := make(map[string]string)
	for _, allocation := range k.GetAllAllocation(ctx) {
		if allocation.Type != structstypes.AllocationType_automated {
			continue
		}

		if allocation.SourceObjectId == "" {
			logger.Error("automated allocation has no source object; cannot index it",
				"allocationId", allocation.Id)
			continue
		}

		// One source can carry only one automated allocation. That is enforced on
		// creation and so should be unreachable, but corrupt state must resolve
		// the same way on every node rather than by map order.
		if owner, taken := claimedBy[allocation.SourceObjectId]; taken {
			logger.Error("two automated allocations claim one source; keeping the first",
				"allocationId", allocation.Id, "sourceObjectId", allocation.SourceObjectId, "keptBy", owner)
			collided++
			continue
		}

		claimedBy[allocation.SourceObjectId] = allocation.Id
		k.SetAutoResizeAllocationSource(ctx, allocation.Id, allocation.SourceObjectId)
		indexed++

		previous, existed := before[allocation.SourceObjectId]
		switch {
		case !existed:
			// Should not happen: every path that creates an automated allocation
			// writes the hook, so a missing one means state this binary did not
			// write.
			added++
		case previous != allocation.Id:
			rekeyed++
		}
	}

	dropped := cleared - (indexed - added)

	// The leak makes a non-zero drop count the expected outcome on any chain that
	// has torn down an automated allocation. Additions and re-keys are the ones
	// that should not happen at all.
	if added > 0 || rekeyed > 0 || collided > 0 {
		logger.Error("v0.21.0 auto-resize index rebuild found more than stale rows",
			"hooksAdded", added, "hooksRekeyed", rekeyed, "sourceCollisions", collided)
	}

	logger.Info("v0.21.0 auto-resize index rebuild complete",
		"rowsBefore", cleared, "rowsAfter", indexed,
		"staleHooksDropped", dropped, "hooksAdded", added,
		"hooksRekeyed", rekeyed, "sourceCollisions", collided)
	return nil
}

// MigrateOrphanedAllocationControllers re-homes allocations whose controller is
// not a player.
//
// AllocationTransfer validated msg.Controller with a guard on
// CurrentContext.GetPlayer, which never returns an error, so the branch could not
// run. Nothing else checked: CanBeTransferBy asks only whether the caller may
// transfer. A transfer to any string therefore committed, because the write is
// keyed by allocation id rather than player id, leaving an allocation controlled
// by somebody who can never sign for it and a permission row keyed to them.
//
// The allocation is not stranded — the source owner keeps PermSourceAllocation on
// the source, which is what AllocationUpdate and AllocationDelete check, and the
// creator keeps the PermAdmin needed to transfer it back — but nobody can connect
// it while it points at a phantom. This hands it to the source owner and clears
// the dead row.
//
// It grants only PermAllocationConnection, so the result is exactly what a
// legitimate transfer to that owner would have produced. It deliberately does not
// mint PermAdmin, and it touches no row but the phantom's: the creator's row on
// the same allocation carries the PermDelete that AllocationDelete falls back to.
//
// Idempotent, and expected to be a no-op: after the handler fix a controller can
// only be a real player, so a re-run finds nothing.
func MigrateOrphanedAllocationControllers(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateOrphanedAllocationControllers")

	k := keepers.StructsKeeper
	cc := k.NewCurrentContext(ctx)

	var (
		rehomed       int
		unrecoverable int
	)

	// GetAllAllocation walks the allocation prefix in key order, so every node
	// repairs the same allocations in the same sequence.
	for _, allocation := range k.GetAllAllocation(ctx) {
		if _, found := k.GetPlayer(ctx, allocation.Controller); found {
			continue
		}

		// GetPermissionedObject returns a cache for any well-formed id, so the
		// owner id it reports has to be confirmed against state in turn.
		var newController string
		if source := cc.GetPermissionedObject(allocation.SourceObjectId); source != nil {
			if candidate := source.GetOwnerId(); candidate != "" {
				if _, found := k.GetPlayer(ctx, candidate); found {
					newController = candidate
				}
			}
		}

		if newController == "" {
			// Leave it rather than guess. The source owner can still update or
			// delete it through PermSourceAllocation on the source.
			logger.Error("allocation controller is not a player and its source has no usable owner; leaving it alone",
				"allocationId", allocation.Id, "controller", allocation.Controller,
				"sourceObjectId", allocation.SourceObjectId)
			unrecoverable++
			continue
		}

		oldController := allocation.Controller
		k.PermissionClearAll(ctx, structskeeper.GetObjectPermissionIDBytes(allocation.Id, oldController))

		newControllerPermissionId := structskeeper.GetObjectPermissionIDBytes(allocation.Id, newController)
		k.SetPermissionsByBytes(ctx, newControllerPermissionId,
			k.GetPermissionsByBytes(ctx, newControllerPermissionId)|structstypes.PermAllocationConnection)

		allocation.Controller = newController
		if _, err := k.SetAllocationOnly(ctx, allocation); err != nil {
			return fmt.Errorf("re-homing allocation %s from %s to %s: %w", allocation.Id, oldController, newController, err)
		}

		logger.Error("allocation was controlled by a player that does not exist; re-homed to its source owner",
			"allocationId", allocation.Id, "previousController", oldController, "newController", newController)
		rehomed++
	}

	if rehomed > 0 || unrecoverable > 0 {
		logger.Error("v0.21.0 orphaned allocation controller repair found damage",
			"rehomed", rehomed, "leftAlone", unrecoverable)
	}

	logger.Info("v0.21.0 orphaned allocation controller repair complete",
		"rehomed", rehomed, "leftAlone", unrecoverable)
	return nil
}

// MigrateInfusionOwnership re-homes infusions whose PlayerId no longer matches
// the player their address belongs to.
//
// UpsertInfusion wrote PlayerId only when creating the record, and nothing else
// ever wrote the field. An address can change hands — AddressRevoke clears the
// address index and AddressRegister binds an unindexed address to any player on
// a key proof — while the infusion, keyed by (destination, address), survives
// intact. Every reactor hook then went on refreshing that record's fuel with
// ownership still pinned to whoever held the address first.
//
// Two things were wrong as a result, and this fixes both at once because they
// are the same field. The delegator's share of the infusion's power was credited
// to the former player's grid capacity, so the new owner staked and got nothing
// while the old one kept capacity they no longer funded. And GuildMembershipJoin
// used the stored PlayerId as its whole ownership check before redelegating on
// infusion.Address's behalf, so the former owner could move the current owner's
// stake.
//
// Registered before MigrateReconcileReactorInfusions so that the reconcile,
// which recomputes each row's fuel and therefore its capacity contribution,
// works against corrected owners.
//
// An address that is no longer registered to anybody is logged and left alone.
// There is nobody to re-home it to, and stripping the capacity would punish the
// current holder of the stake for a bug. The row heals on its own the next time
// the address is registered and staking touches it.
//
// Idempotent, and expected to be a no-op after the first run: SetPlayerId
// returns without writing when the id already matches.
func MigrateInfusionOwnership(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateInfusionOwnership")

	k := keepers.StructsKeeper
	cc := k.NewCurrentContext(ctx)

	var (
		visited       int
		rehomed       int
		unregistered  int
		capacityMoved uint64
	)

	// GetAllInfusion walks the infusion prefix in key order, so every node
	// repairs the same rows in the same sequence. It covers struct generator
	// infusions as well as reactor ones, which carry the same field and took
	// the same misattribution.
	for _, infusion := range k.GetAllInfusion(ctx) {
		visited++

		playerIndex := k.GetPlayerIndexFromAddress(ctx, infusion.Address)
		if playerIndex == 0 {
			// Not an error in the record: the address is simply unregistered.
			logger.Error("infusion address belongs to no player; leaving ownership alone",
				"infusionId", infusion.DestinationId+"-"+infusion.Address,
				"address", infusion.Address, "playerId", infusion.PlayerId)
			unregistered++
			continue
		}

		currentPlayerId := structskeeper.GetObjectID(structstypes.ObjectType_player, playerIndex)
		if currentPlayerId == infusion.PlayerId {
			continue
		}

		_, _, playerPower := infusion.GetPowerDistribution()

		cc.GetInfusion(infusion.DestinationId, infusion.Address).SetPlayerId(currentPlayerId)

		rehomed++
		capacityMoved += playerPower

		logger.Error("infusion ownership was stale; re-homed to the address's current player",
			"infusionId", infusion.DestinationId+"-"+infusion.Address,
			"address", infusion.Address, "previousPlayerId", infusion.PlayerId,
			"newPlayerId", currentPlayerId, "capacity", playerPower)
	}

	cc.CommitAll()

	logger.Info("v0.21.0 infusion ownership repair complete",
		"infusionsVisited", visited, "rehomed", rehomed,
		"capacityMoved", capacityMoved, "unregisteredAddresses", unregistered)

	return nil
}

/* MigrateDelegationDistributionState repairs delegations that the pre-v0.21.0
 * address sweep left unable to touch their own rewards.
 *
 * The old sweep was a RemoveDelegation followed by a SetDelegation, and neither
 * call maintained distribution. SetDelegation is a bare store write, so the
 * destination never received the DelegatorStartingInfo that prices its
 * rewards. And distribution's BeforeDelegationRemoved is a no-op -- the
 * withdrawal lives on BeforeDelegationSharesModified, which a removal does not
 * fire -- so the source's record was not settled either; it was simply
 * abandoned, still holding its reference on the validator's historical rewards
 * for that period. The result is a matched pair of broken rows: a delegation
 * that can never withdraw, undelegate or redelegate (every attempt fails on
 * ErrEmptyDelegationDistInfo, permanently, which is why its shares have been
 * frozen ever since), and a starting info that nothing can ever claim.
 *
 * Where the pair can be identified, re-homing the abandoned record onto the
 * delegation that inherited its stake repairs both at once, and is exact rather
 * than approximate: the period it records is the period that stake really was
 * delegated at, so the rewards it accrued across the move are paid to whoever
 * holds the stake now. It also needs no reference-count adjustment, one record
 * continuing to hold the one reference.
 *
 * Identification is the limit. Nothing on disk links an abandoned record to the
 * address its stake went to, so the pairing is only made where a validator has
 * exactly one of each, which is the shape a single address move leaves. Where
 * it does not, the delegation is initialized fresh at the current period and the
 * abandoned record is left in place. Both halves of that fallback are the
 * conservative choice. A fresh period forfeits the span since the move, but
 * inventing an earlier one would pay this delegator out of rewards belonging to
 * everyone else delegating to that validator, and under-crediting one player is
 * recoverable where draining a reward pool is not. Leaving the record beats
 * deleting it, because decrementReferenceCount is not exported: a delete would
 * pin the same historical rewards forever *and* destroy the evidence needed to
 * re-home it later.
 *
 * Idempotent. A second pass finds no delegation missing starting info, so it
 * re-homes nothing and initializes nothing; it only re-logs any record it could
 * not pair.
 */
func MigrateDelegationDistributionState(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateDelegationDistributionState")

	distrHooks := keepers.DistrKeeper.Hooks()

	abandoned, err := abandonedStartingInfosByValidator(ctx, keepers)
	if err != nil {
		return err
	}

	validators, err := keepers.StakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return err
	}

	var (
		visited     int
		rehomed     int
		initialized int
		unpaired    int
	)

	// GetAllValidators walks the validator prefix in key order and
	// GetValidatorDelegations does the same within each, so every node repairs
	// the same pairs in the same sequence -- which matters, because
	// initializing a delegation increments the validator's period.
	for _, validator := range validators {
		valAddr, valAddrErr := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if valAddrErr != nil {
			logger.Error("unparsable validator operator address; skipping",
				"operatorAddress", validator.OperatorAddress, "error", valAddrErr)
			continue
		}

		delegations, delegationsErr := keepers.StakingKeeper.GetValidatorDelegations(ctx, valAddr)
		if delegationsErr != nil {
			return delegationsErr
		}

		var needy []sdk.AccAddress
		for _, delegation := range delegations {
			visited++

			delAddr, delAddrErr := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
			if delAddrErr != nil {
				logger.Error("unparsable delegator address; skipping",
					"delegatorAddress", delegation.DelegatorAddress, "error", delAddrErr)
				continue
			}

			hasStartingInfo, hasErr := keepers.DistrKeeper.HasDelegatorStartingInfo(ctx, valAddr, delAddr)
			if hasErr != nil {
				return hasErr
			}
			if !hasStartingInfo {
				needy = append(needy, delAddr)
			}
		}

		orphans := abandoned[validator.OperatorAddress]

		if len(needy) == 1 && len(orphans) == 1 {
			source := orphans[0]

			info, infoErr := keepers.DistrKeeper.GetDelegatorStartingInfo(ctx, valAddr, source)
			if infoErr != nil {
				return infoErr
			}
			if err := keepers.DistrKeeper.SetDelegatorStartingInfo(ctx, valAddr, needy[0], info); err != nil {
				return err
			}
			if err := keepers.DistrKeeper.DeleteDelegatorStartingInfo(ctx, valAddr, source); err != nil {
				return err
			}

			rehomed++

			logger.Error("re-homed an abandoned delegator starting info onto the delegation that inherited its stake",
				"validator", validator.OperatorAddress, "from", source.String(), "to", needy[0].String(),
				"previousPeriod", info.PreviousPeriod, "stake", info.Stake.String())

			continue
		}

		for _, delAddr := range needy {
			// The same pair of calls the transfer makes, for the same reason:
			// initializeDelegation prices from Period-1, so the period has to
			// have moved first.
			if err := distrHooks.BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
				return err
			}
			if err := distrHooks.AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
				return err
			}

			initialized++

			logger.Error("delegation had no distribution starting info and no unambiguous predecessor; initialized at the current period",
				"validator", validator.OperatorAddress, "delegator", delAddr.String(),
				"abandonedRecordsOnValidator", len(orphans))
		}

		for _, source := range orphans {
			unpaired++
			logger.Error("delegator starting info has no delegation behind it and could not be paired; left in place",
				"validator", validator.OperatorAddress, "delegator", source.String())
		}
	}

	logger.Info("v0.21.0 delegation distribution state repair complete",
		"delegationsVisited", visited, "startingInfosRehomed", rehomed,
		"delegationsInitialized", initialized, "abandonedRecordsLeftInPlace", unpaired)

	return nil
}

// abandonedStartingInfosByValidator collects every DelegatorStartingInfo with
// no delegation behind it, grouped by validator. IterateDelegatorStartingInfos
// walks in key order, so the slices are ordered identically on every node.
func abandonedStartingInfosByValidator(ctx context.Context, keepers *upgrades.Keepers) (map[string][]sdk.AccAddress, error) {
	abandoned := make(map[string][]sdk.AccAddress)

	var iterationErr error
	keepers.DistrKeeper.IterateDelegatorStartingInfos(ctx,
		func(val sdk.ValAddress, del sdk.AccAddress, _ distrtypes.DelegatorStartingInfo) bool {
			if _, err := keepers.StakingKeeper.GetDelegation(ctx, del, val); err == nil {
				return false
			} else if !errors.Is(err, stakingtypes.ErrNoDelegation) {
				iterationErr = err
				return true
			}

			abandoned[val.String()] = append(abandoned[val.String()], del)
			return false
		})

	return abandoned, iterationErr
}

/* MigrateAuditDelegatorShares reports, and deliberately does not repair, stake
 * that the pre-v0.21.0 address sweep may have knocked out of agreement with the
 * validator it belongs to.
 *
 * Two ways it could happen, both from the same bare SetDelegation. If the
 * destination address already delegated to that validator, the write replaced
 * its record instead of merging, so shares went missing while
 * Validator.DelegatorShares kept counting them. And if a delegation was swept
 * twice, the second RemoveDelegation failed inside distribution's hook -- the
 * first sweep having left no starting info to withdraw against -- and the old
 * code discarded that error and wrote the destination anyway, duplicating the
 * shares.
 *
 * Neither is repairable here, and the reason is worth stating rather than
 * assuming. The discrepancy is known per validator but not per delegator:
 * nothing on disk records which address lost shares or which gained them.
 * Adjusting DelegatorShares to match would silently reprice every other
 * delegation to that validator, and adjusting a delegation would mint or burn
 * one player's stake on a guess. Both are worse than an accurate log, so this
 * reports at error level with the exact numbers and leaves state alone for a
 * human to settle.
 *
 * Read-only, therefore trivially idempotent, and silent on a chain where no
 * address ever swept a delegation.
 */
func MigrateAuditDelegatorShares(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateAuditDelegatorShares")

	validators, err := keepers.StakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return err
	}

	var (
		visited    int
		discrepant int
	)

	for _, validator := range validators {
		visited++

		valAddr, valAddrErr := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if valAddrErr != nil {
			logger.Error("unparsable validator operator address; skipping",
				"operatorAddress", validator.OperatorAddress, "error", valAddrErr)
			continue
		}

		delegations, delegationsErr := keepers.StakingKeeper.GetValidatorDelegations(ctx, valAddr)
		if delegationsErr != nil {
			return delegationsErr
		}

		total := math.LegacyZeroDec()
		for _, delegation := range delegations {
			total = total.Add(delegation.Shares)
		}

		if total.Equal(validator.DelegatorShares) {
			continue
		}

		discrepant++

		logger.Error("validator delegator shares disagree with the delegations behind them; NOT repaired",
			"validator", validator.OperatorAddress,
			"delegatorShares", validator.DelegatorShares.String(),
			"delegationsTotal", total.String(),
			"difference", validator.DelegatorShares.Sub(total).String(),
			"delegationCount", len(delegations),
			"direction", sharesDiscrepancyDirection(validator.DelegatorShares, total))
	}

	if discrepant == 0 {
		logger.Info("v0.21.0 delegator share audit found no discrepancies", "validatorsVisited", visited)
	} else {
		logger.Error("v0.21.0 delegator share audit found discrepancies requiring manual settlement",
			"validatorsVisited", visited, "validatorsDiscrepant", discrepant)
	}

	return nil
}

// sharesDiscrepancyDirection names which of the two pre-v0.21.0 sweep bugs a
// discrepancy looks like, since they point opposite ways.
func sharesDiscrepancyDirection(delegatorShares, delegationsTotal math.LegacyDec) string {
	if delegatorShares.GT(delegationsTotal) {
		return "orphaned_shares"
	}
	return "duplicated_shares"
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
