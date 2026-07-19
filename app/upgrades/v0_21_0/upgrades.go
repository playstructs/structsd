package v0_21_0

import (
	"context"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
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
		changed := false
		if guild.BankConvertInFee.IsNil() {
			guild.BankConvertInFee = math.LegacyZeroDec()
			changed = true
		}
		if guild.BankConvertOutFee.IsNil() {
			guild.BankConvertOutFee = math.LegacyZeroDec()
			changed = true
		}
		if changed {
			k.SetGuild(ctx, guild)
			guildsMigrated++
		}
	}

	logger.Info("v0.21.0 guild bank fee backfill complete", "guildsMigrated", guildsMigrated)
	return nil
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
