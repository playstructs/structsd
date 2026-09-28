package types_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestAssetOwnerIdentityCanonicalAndTyped(t *testing.T) {
	for _, id := range []string{"1-1", "0-1"} {
		kind, index, err := types.ParseAssetOwnerID(id)
		require.NoError(t, err)
		require.Equal(t, uint64(1), index)
		require.True(t, kind == types.ObjectType_player || kind == types.ObjectType_guild)
	}
	for _, id := range []string{"", "1", "1-0", "1-01", "01-1", "1-+1", "1-1-1", "9-1", "1-18446744073709551616"} {
		_, _, err := types.ParseAssetOwnerID(id)
		require.Error(t, err, id)
	}
}

func TestGuildRankPriceCurvesAndBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                                          string
		entry, current, target, base, increment, want uint64
	}{
		{"flat ninety ranks", 101, 101, 11, 1_000_000, 0, 90_000_000},
		{"increasing ninety ranks", 101, 101, 11, 1_000_000, 100_000, 490_500_000},
		{"free", 101, 101, 1, 0, 0, 0},
		{"demoted below entry", 101, 110, 100, 1_000_000, 0, 1_000_000},
		{"large totals small difference", math.MaxUint64, 2, 1, 2, 0, 2},
		{"largest payable", 2, 2, 1, math.MaxUint64, 0, math.MaxUint64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := types.CalculateGuildRankPurchasePrice(tc.entry, tc.current, tc.target, tc.base, tc.increment)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
	for _, ranks := range [][3]uint64{{0, 2, 1}, {2, 0, 1}, {2, 2, 0}, {101, 10, 10}, {101, 10, 11}, {101, 110, 105}} {
		_, err := types.CalculateGuildRankPurchasePrice(ranks[0], ranks[1], ranks[2], 1, 0)
		require.Error(t, err)
	}
	_, err := types.CalculateGuildRankPurchasePrice(3, 3, 1, math.MaxUint64, math.MaxUint64)
	require.Error(t, err)
}

func TestGuildRankPriceBatchingDoesNotDiscount(t *testing.T) {
	for base := uint64(0); base < 4; base++ {
		for increment := uint64(0); increment < 4; increment++ {
			whole, err := types.CalculateGuildRankPurchasePrice(101, 101, 11, base, increment)
			require.NoError(t, err)
			var sum uint64
			for rank := uint64(101); rank > 11; rank-- {
				step, err := types.CalculateGuildRankPurchasePrice(101, rank, rank-1, base, increment)
				require.NoError(t, err)
				sum += step
			}
			require.Equal(t, whole, sum)
		}
	}
}
