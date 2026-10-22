package v0_22_0_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestMigrateStructBuildCharge(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	k.SetStructType(ctx, types.StructType{Id: 1, BuildCharge: 8, MaxHealth: 7, BuildLimit: 1})
	k.SetStructType(ctx, types.StructType{Id: 2, BuildCharge: 1, BuildLimit: 1})

	require.NoError(t, v22.MigrateStructBuildCharge(ctx, keepers))

	for _, id := range []uint64{1, 2} {
		structType, found := k.GetStructType(ctx, id)
		require.True(t, found)
		require.Equal(t, v22.StructBuildCharge, structType.BuildCharge)
	}
	structType, _ := k.GetStructType(ctx, 1)
	require.Equal(t, uint64(7), structType.MaxHealth, "other fields untouched")
}

func TestGenesisStructTypesUseUpgradeBuildCharge(t *testing.T) {
	for _, structType := range types.CreateStructTypeGenesis() {
		require.Equal(t, v22.StructBuildCharge, structType.BuildCharge, "struct type %d", structType.Id)
	}
}
