package v0_21_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
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

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
