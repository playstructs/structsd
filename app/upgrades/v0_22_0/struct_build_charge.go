package v0_22_0

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/app/upgrades"
)

// StructBuildCharge is the build charge every struct type carries from v0.22.0.
const StructBuildCharge uint64 = 1

// MigrateStructBuildCharge sets BuildCharge on every stored struct type. Only
// that field is written, so any other struct-type value in state is preserved
// rather than reset from genesis. StructTrash gates on the same field.
func MigrateStructBuildCharge(ctx context.Context, keepers *upgrades.Keepers) error {
	k := keepers.StructsKeeper
	var changed int
	for _, structType := range k.GetAllStructType(ctx) {
		if structType.BuildCharge == StructBuildCharge {
			continue
		}
		structType.BuildCharge = StructBuildCharge
		k.SetStructType(ctx, structType)
		changed++
	}
	sdk.UnwrapSDKContext(ctx).Logger().Info("struct build charge set",
		"upgrade", UpgradeName, "buildCharge", StructBuildCharge, "structTypesChanged", changed)
	return nil
}
