package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

/* Regression suite for the unbounded expiration bucket.
 *
 * AgreementExpirations reads the index at exactly the current height and tears
 * down everything it finds, inside the EndBlocker, which has no gas meter. Each
 * entry checkpoints its provider, moves money, destroys an allocation and
 * rewrites indexes. The consumer picks the duration and therefore the height, so
 * agreements opened across many earlier blocks can be aimed at one block.
 *
 * The bound is on creating that work rather than on doing it, and that is forced
 * rather than chosen. A grid cascade can stop half way and resume next block; an
 * expiry cannot, because the index is read at one exact height and an agreement
 * missed there is never revisited - meanwhile its capacity stays in the
 * provider's aggregate load and Checkpoint() bills it out of other consumers'
 * escrow. So a full height refuses new arrivals instead.
 */

// fillExpirationHeight puts count agreements into the expiration bucket for
// block, writing the index directly. The point under test is the bucket's
// cardinality, not how the entries got there, and building 256 real agreements
// would need 256 units of grid capacity.
func fillExpirationHeight(t *testing.T, f *teardownFixture, block uint64, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		require.NoError(t, f.k.SetAgreementExpirationIndex(f.ctx, block, fmt.Sprintf("filler-%d", i)))
	}
	require.Len(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, block), count)
}

func TestAgreementExpirationBucket_HasRoomUntilTheCap(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")
	const block = uint64(9_000)

	fillExpirationHeight(t, f, block, types.AgreementExpirationBucketCap-1)
	require.True(t, f.k.AgreementExpirationHeightHasRoomFor(f.ctx, block, ""),
		"one slot short of the cap must still admit an agreement")

	require.NoError(t, f.k.SetAgreementExpirationIndex(f.ctx, block, "one-more"))
	require.False(t, f.k.AgreementExpirationHeightHasRoomFor(f.ctx, block, ""),
		"a full height must refuse a new agreement")
}

/* TestAgreementExpirationBucket_AlreadyIndexedAgreementKeepsItsSlot is what
 * stops the cap breaking a re-price. A capacity change re-bases the window and
 * can land back on the block the agreement already expires at; that adds no work
 * to the EndBlocker, so refusing it would be a rejection for nothing.
 */
func TestAgreementExpirationBucket_AlreadyIndexedAgreementKeepsItsSlot(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")
	const block = uint64(9_000)

	fillExpirationHeight(t, f, block, types.AgreementExpirationBucketCap)
	require.False(t, f.k.AgreementExpirationHeightHasRoomFor(f.ctx, block, "outsider"))
	require.True(t, f.k.AgreementExpirationHeightHasRoomFor(f.ctx, block, "filler-0"),
		"an agreement already in the bucket occupies a slot it need not take twice")
}

/* TestAgreementExpirationBucket_CheckIsBoundedOnAnOverfullHeight covers the
 * chain upgrading into this rule with buckets already past the cap. The check
 * counts rather than reading a maintained counter - so there is nothing to
 * migrate and nothing to drift - and it stops at the cap, which is what keeps
 * counting affordable when the bucket is enormous.
 */
func TestAgreementExpirationBucket_CheckIsBoundedOnAnOverfullHeight(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")
	const block = uint64(9_000)

	fillExpirationHeight(t, f, block, types.AgreementExpirationBucketCap*3)

	require.False(t, f.k.AgreementExpirationHeightHasRoomFor(f.ctx, block, ""),
		"an over-full height accepts nothing new until it drains")
}

// TestAgreementOpen_RefusesAFullExpirationHeight is the handler-level
// regression: the refusal has to happen before the collateral moves, not be
// unwound by the transaction rolling back.
func TestAgreementOpen_RefusesAFullExpirationHeight(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")
	tightenProvider(t, f, 1, 10000, 1, 1000000)

	const duration = uint64(50)
	targetBlock := uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight()) + duration
	fillExpirationHeight(t, f, targetBlock, types.AgreementExpirationBucketCap)

	collateral := int64(100 * duration * 10)
	f.fund(t, f.consumerAcc, collateral)
	balanceBefore := f.balance(f.consumerAcc)

	_, err := f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{
		Creator:    f.consumer.Creator,
		ProviderId: f.provider.Id,
		Capacity:   100,
		Duration:   duration,
	})
	require.Error(t, err, "a full expiration height must refuse the agreement")
	require.ErrorContains(t, err, "too many agreements")

	require.Equal(t, balanceBefore, f.balance(f.consumerAcc),
		"a refused open must not have taken collateral")

	// One block later the target height is a different, empty one.
	advanceBlocks(f, 1)
	_, err = f.ms.AgreementOpen(f.ctx, &types.MsgAgreementOpen{
		Creator:    f.consumer.Creator,
		ProviderId: f.provider.Id,
		Capacity:   100,
		Duration:   duration,
	})
	require.NoError(t, err, "shifting off the congested height must work")
}

/* TestDurationIncrease_RefusesAFullExpirationHeight covers the other way an
 * agreement reaches a height: moving there. An extension is indistinguishable
 * from an open once it is indexed, so it is held to the same cap.
 */
func TestDurationIncrease_RefusesAFullExpirationHeight(t *testing.T) {
	f, agreement := durationFixture(t)

	stored, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)

	const extension = uint64(25)
	fillExpirationHeight(t, f, stored.EndBlock+extension, types.AgreementExpirationBucketCap)

	require.Error(t, increaseDuration(f, agreement.Id, extension),
		"an extension onto a full height must be refused")
	requireExpirationIntact(t, f, agreement.Id, stored.EndBlock)

	// A neighbouring height is free.
	require.NoError(t, increaseDuration(f, agreement.Id, extension+1))
}
