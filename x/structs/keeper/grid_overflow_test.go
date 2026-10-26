package keeper_test

import (
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* Regression suite for unguarded addition on grid attributes.
 *
 * SetGridAttributeIncrement and the addition inside SetGridAttributeDelta were
 * the only arithmetic here without a guard - both siblings already clamped their
 * subtraction, which is what made these the odd ones out rather than a
 * considered choice.
 *
 * Wrapping is the worst of the outcomes available: capacity, load, stored ore
 * and the replay nonces are all treated as monotonic by their callers, so a roll
 * over to a small number destroys accounted value, understates committed load,
 * and in the nonce case resets replay protection. Saturation is corrupt state
 * too, but bounded and in a known direction.
 *
 * Note what this does NOT claim: the bound is not reachable today. The only
 * compounding path is a cycle of allocations feeding each other, and that grows
 * linearly. This is the backstop.
 */

func gridOverflowFixture(t *testing.T) (keeperlib.Keeper, sdk.Context, string) {
	t.Helper()

	k, ctx := keepertest.StructsKeeper(t)
	attributeId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, "4-1")

	return k, ctx, attributeId
}

func TestGridAttributeIncrement_SaturatesInsteadOfWrapping(t *testing.T) {
	k, ctx, attributeId := gridOverflowFixture(t)

	cc := k.NewCurrentContext(ctx)
	cc.SetGridAttribute(attributeId, math.MaxUint64-10)

	result := cc.SetGridAttributeIncrement(attributeId, 100)
	require.Equal(t, uint64(math.MaxUint64), result,
		"an increment past the top must clamp, not roll over to 89")
	require.Equal(t, uint64(math.MaxUint64), cc.GetGridAttribute(attributeId))

	cc.CommitAll()
	require.Equal(t, uint64(math.MaxUint64), k.GetGridAttribute(ctx, attributeId),
		"the clamped value is what gets committed")
}

// An ordinary increment must be untouched by the guard.
func TestGridAttributeIncrement_NormalAdditionUnchanged(t *testing.T) {
	k, ctx, attributeId := gridOverflowFixture(t)

	cc := k.NewCurrentContext(ctx)
	cc.SetGridAttribute(attributeId, 500)

	require.Equal(t, uint64(650), cc.SetGridAttributeIncrement(attributeId, 150))
	require.Equal(t, uint64(650), cc.GetGridAttribute(attributeId))
}

// Exactly reaching MaxUint64 is not an overflow and must not be logged as one.
func TestGridAttributeIncrement_ExactMaximumIsNotOverflow(t *testing.T) {
	k, ctx, attributeId := gridOverflowFixture(t)

	cc := k.NewCurrentContext(ctx)
	cc.SetGridAttribute(attributeId, math.MaxUint64-10)

	require.Equal(t, uint64(math.MaxUint64), cc.SetGridAttributeIncrement(attributeId, 10))
}

/* SetGridAttributeDelta guarded its subtraction and not its addition, so it
 * carried the same defect one line down. The report named only Increment.
 */
func TestGridAttributeDelta_SaturatesInsteadOfWrapping(t *testing.T) {
	k, ctx, attributeId := gridOverflowFixture(t)

	cc := k.NewCurrentContext(ctx)
	cc.SetGridAttribute(attributeId, math.MaxUint64-10)

	// Remove nothing, add a lot: resetAmount stays near the top and the addition
	// is what rolls over.
	result := cc.SetGridAttributeDelta(attributeId, 0, 100)
	require.Equal(t, uint64(math.MaxUint64), result,
		"the addition inside Delta must clamp too")
	require.Equal(t, uint64(math.MaxUint64), cc.GetGridAttribute(attributeId))
}

func TestGridAttributeDelta_NormalReplacementUnchanged(t *testing.T) {
	k, ctx, attributeId := gridOverflowFixture(t)

	cc := k.NewCurrentContext(ctx)
	cc.SetGridAttribute(attributeId, 500)

	// Swap a 200 contribution for a 50 one.
	require.Equal(t, uint64(350), cc.SetGridAttributeDelta(attributeId, 200, 50))
}
