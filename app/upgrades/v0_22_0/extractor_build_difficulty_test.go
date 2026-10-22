package v0_22_0_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"structs/app/upgrades"
	v22 "structs/app/upgrades/v0_22_0"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func TestMigrateExtractorBuildDifficulty(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	want := types.CreateStructTypeGenesis()
	for _, original := range want {
		if original.Id == 14 {
			original.BuildDifficulty = 700
			original.MaxHealth = 17
			original.OreMiningDifficulty = 12345
		}
		k.SetStructType(ctx, original)
	}
	for i := range want {
		if want[i].Id == 14 {
			want[i].BuildDifficulty = v22.ExtractorBuildDifficulty
			want[i].MaxHealth = 17
			want[i].OreMiningDifficulty = 12345
		}
	}

	require.NoError(t, v22.MigrateExtractorBuildDifficulty(ctx, keepers))
	require.Equal(t, want, k.GetAllStructType(ctx), "only extractor build difficulty changes")
	eventCount := len(ctx.EventManager().Events())
	require.NoError(t, v22.MigrateExtractorBuildDifficulty(ctx, keepers))
	require.Equal(t, want, k.GetAllStructType(ctx))
	require.Len(t, ctx.EventManager().Events(), eventCount, "rerun should not rewrite the type")
}

func TestMigrateExtractorBuildDifficultyMissingType(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	require.NoError(t, v22.MigrateExtractorBuildDifficulty(ctx, &upgrades.Keepers{StructsKeeper: k}))
	_, found := k.GetStructType(ctx, 14)
	require.False(t, found, "migration must not invent an absent type")
}

func TestExtractorBuildDifficultyTargetsOneMinute(t *testing.T) {
	var extractor types.StructType
	for _, structType := range types.CreateStructTypeGenesis() {
		if structType.Id == 14 {
			extractor = structType
		}
	}
	require.Equal(t, uint64(14), extractor.Id)
	require.Equal(t, uint64(10), extractor.BuildDifficulty)
	require.Equal(t, v22.ExtractorBuildDifficulty, extractor.BuildDifficulty)
	require.Equal(t, uint64(60), extractor.BuildDifficulty*6)
	for _, tc := range []struct {
		age  uint64
		want int
	}{{0, 64}, {8, 8}, {9, 4}, {10, 1}, {11, 1}} {
		require.Equal(t, tc.want, types.CalculateDifficulty(tc.age, extractor.BuildDifficulty), "age %d", tc.age)
	}
}
