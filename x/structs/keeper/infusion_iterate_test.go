package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* IterateInfusionsByDestination exists so an upgrade migration does not have to
 * hold a destination's whole infusion set in memory at once.
 *
 * The work an upgrade does is unavoidable - it runs at a coordinated height with
 * an infinite gas meter, against live cardinality - but holding all of it at once
 * is not, and exhausting memory during an upgrade block is a worse failure than a
 * slow one: the block is retried from the same state.
 *
 * The property under test is that streaming and collecting see exactly the same
 * rows in the same order. A streaming variant that skipped or reordered rows
 * would corrupt whatever migration used it, silently.
 */
func seedDestinationInfusions(t *testing.T, k keeperlib.Keeper, ctx sdk.Context, destinationId string, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		k.SetInfusion(ctx, types.Infusion{
			DestinationType: types.ObjectType_reactor,
			DestinationId:   destinationId,
			Address:         sdk.AccAddress(fmt.Sprintf("iterinfusion%023d", i)).String(),
			Fuel:            uint64(i + 1),
		})
	}
}

func TestIterateInfusionsByDestination_MatchesCollectedWalk(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	const destinationId = "3-1"

	seedDestinationInfusions(t, k, ctx, destinationId, 40)

	collected := k.GetAllInfusionsByDestination(ctx, destinationId)
	require.Len(t, collected, 40)

	var streamed []types.Infusion
	k.IterateInfusionsByDestination(ctx, destinationId, func(infusion types.Infusion) {
		streamed = append(streamed, infusion)
	})

	require.Equal(t, collected, streamed,
		"streaming must visit exactly the rows a collected walk does, in the same order")
}

func TestIterateInfusionsByDestination_EmptyAndUnrelated(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	seedDestinationInfusions(t, k, ctx, "3-1", 5)

	var visited int
	k.IterateInfusionsByDestination(ctx, "3-2", func(types.Infusion) { visited++ })
	require.Zero(t, visited, "a destination with no infusions must visit nothing")

	k.IterateInfusionsByDestination(ctx, "3-1", func(infusion types.Infusion) {
		require.Equal(t, "3-1", infusion.DestinationId,
			"the prefix must not leak rows from another destination")
		visited++
	})
	require.Equal(t, 5, visited)
}
