package keeper_test

import (
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

/* Regression suite for DurationIncrease's arithmetic.
 *
 * Both the duration and the end block were computed with unchecked uint64
 * addition, and AgreementDurationVerify was handed the already-wrapped result -
 * so it validated a number that had nothing to do with what was about to be
 * stored. The reported shape: StartBlock 100, EndBlock 110, amount 2^64-5 gives
 * a "duration" of 5, which passes any sane published range, and an EndBlock of
 * 105.
 *
 * The consequence is not a wrong number, it is an agreement that never ends.
 * Commit re-indexes the expiration at the stored EndBlock, and AgreementExpirations
 * reads the index at *exactly* the current height - no range scan, no retry - so
 * an index written in the past is never visited. The agreement keeps its
 * allocation and its capacity in the provider's load, and Checkpoint() goes on
 * billing that capacity against the shared collateral pool every block, out of
 * other consumers' escrow.
 */

// durationFixture reuses the teardown fixture's funded consumer and open-market
// provider, and widens the published duration range so nothing but the
// arithmetic under test can reject a change.
func durationFixture(t *testing.T) (*teardownFixture, types.Agreement) {
	t.Helper()

	f := setupTeardownFixture(t, 10, "0.5", "0.25")
	agreement, _ := f.openAgreement(t, 100, 50)
	tightenProvider(t, f, 1, 10000, 1, math.MaxUint64)

	return f, agreement
}

func increaseDuration(f *teardownFixture, agreementId string, amount uint64) error {
	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	if err := cc.GetAgreement(agreementId).DurationIncrease(amount); err != nil {
		return err
	}
	cc.CommitAll()
	return nil
}

// requireExpirationIntact asserts the agreement still ends where it did and is
// still reachable by the one block that will expire it.
func requireExpirationIntact(t *testing.T, f *teardownFixture, agreementId string, endBlock uint64) {
	t.Helper()

	stored, found := f.k.GetAgreement(f.ctx, agreementId)
	require.True(t, found)
	require.Equal(t, endBlock, stored.EndBlock, "the end block moved")
	require.Contains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, endBlock), agreementId,
		"the expiration index no longer names this agreement at its end block")
}

/* TestDurationIncrease_WrappingAmountRejected is the direct regression. The
 * amount is chosen so the unchecked addition would wrap the end block back into
 * the past while presenting a small, plausible duration to the verifier.
 */
func TestDurationIncrease_WrappingAmountRejected(t *testing.T) {
	f, agreement := durationFixture(t)

	stored, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)

	// Land the wrapped end block five blocks past the start, which is the
	// report's own construction: a duration the provider would happily accept.
	wrapping := math.MaxUint64 - stored.EndBlock + stored.StartBlock + 5 + 1

	err := increaseDuration(f, agreement.Id, wrapping)
	require.Error(t, err, "an amount that wraps the end block must be refused")
	require.ErrorContains(t, err, "duration")

	requireExpirationIntact(t, f, agreement.Id, stored.EndBlock)
}

// TestDurationIncrease_MaxUint64Rejected is the blunt version: the largest value
// the protobuf field can carry. There is no ValidateBasic bounding it.
func TestDurationIncrease_MaxUint64Rejected(t *testing.T) {
	f, agreement := durationFixture(t)

	stored, _ := f.k.GetAgreement(f.ctx, agreement.Id)

	require.Error(t, increaseDuration(f, agreement.Id, math.MaxUint64))
	requireExpirationIntact(t, f, agreement.Id, stored.EndBlock)
}

// TestDurationIncrease_ZeroRejected keeps a no-op from re-indexing and paying.
func TestDurationIncrease_ZeroRejected(t *testing.T) {
	f, agreement := durationFixture(t)

	stored, _ := f.k.GetAgreement(f.ctx, agreement.Id)

	err := increaseDuration(f, agreement.Id, 0)
	require.Error(t, err)
	require.ErrorContains(t, err, "would do nothing")
	requireExpirationIntact(t, f, agreement.Id, stored.EndBlock)
}

/* TestDurationIncrease_InvertedWindowRejected covers an agreement whose end
 * block already precedes its start. No transaction produces one - but a genesis
 * file assigns a whole Agreement record, and the subtraction that derives the
 * duration would wrap on it, handing the verifier a near-2^64 number.
 */
func TestDurationIncrease_InvertedWindowRejected(t *testing.T) {
	f, agreement := durationFixture(t)

	corrupt, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	corrupt.StartBlock = corrupt.EndBlock + 10
	_, err := f.k.SetAgreement(f.ctx, corrupt)
	require.NoError(t, err)

	require.Error(t, increaseDuration(f, agreement.Id, 5),
		"a window that runs backwards must be refused rather than wrapped")
}

/* TestDurationIncrease_ValidExtensionMovesTheIndex is the positive control. It
 * is what makes the rejections above meaningful: the guards must refuse the
 * wrapping cases without refusing an ordinary extension, and the expiration
 * index has to follow the end block or the agreement stops expiring for the
 * opposite reason.
 */
func TestDurationIncrease_ValidExtensionMovesTheIndex(t *testing.T) {
	f, agreement := durationFixture(t)

	stored, _ := f.k.GetAgreement(f.ctx, agreement.Id)
	oldEnd := stored.EndBlock

	require.NoError(t, increaseDuration(f, agreement.Id, 25))

	requireExpirationIntact(t, f, agreement.Id, oldEnd+25)
	require.NotContains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, oldEnd), agreement.Id,
		"the stale index row must be cleared, or the agreement expires early")
}
