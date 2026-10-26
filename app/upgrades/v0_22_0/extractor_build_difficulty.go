package v0_22_0

import (
	"context"

	"structs/app/upgrades"
)

// ExtractorBuildDifficulty reaches minimum proof difficulty after ten blocks:
// approximately one minute at six-second blocks. Valid proofs can finish sooner.
const ExtractorBuildDifficulty uint64 = 10

// MigrateExtractorBuildDifficulty changes only the Ore Extractor's build range.
// Existing builds use the updated type while retaining their original start block.
func MigrateExtractorBuildDifficulty(ctx context.Context, keepers *upgrades.Keepers) error {
	k := keepers.StructsKeeper
	extractor, found := k.GetStructType(ctx, 14)
	if !found || extractor.BuildDifficulty == ExtractorBuildDifficulty {
		return nil
	}
	extractor.BuildDifficulty = ExtractorBuildDifficulty
	k.SetStructType(ctx, extractor)
	return nil
}
