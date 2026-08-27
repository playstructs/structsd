package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

/* Regression suite for a zero-height restart extending live agreements.
 *
 * StartBlock and EndBlock are absolute heights on the agreement record, so an
 * export carries them through untouched. For a height-preserving restart that is
 * exactly right. For a zero-height one it is a gift: an agreement exported at
 * height H with E-H blocks left is re-indexed to expire at E on a chain that
 * restarts near 1, so it keeps supplying capacity for roughly the whole age of
 * the old chain and nobody posted collateral for any of it.
 *
 * The provider's checkpoint has to move with the windows. Checkpoint() bills
 * aggregate load from the checkpoint block, so leaving one clock absolute while
 * the other is rebased bills a span the agreements no longer claim - in this
 * direction it reads as checkpointBlock >= currentBlock and bills nothing at all
 * until the new chain grows past the old height.
 */

func TestZeroHeightRebase_PreservesOnlyRemainingDuration(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	const duration = uint64(50)
	agreement, _ := f.openAgreement(t, 100, duration)

	// Run the chain on a while, so the absolute heights are far from zero and a
	// failure to rebase is unmistakable.
	advanceBlocks(f, 20)
	exportHeight := uint64(sdk.UnwrapSDKContext(f.ctx).BlockHeight())

	stored, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	remaining := stored.EndBlock - exportHeight
	require.Equal(t, uint64(30), remaining, "fixture sanity: 30 blocks should be left")

	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	require.NoError(t, cc.RebaseAgreementsForZeroHeightGenesis())
	cc.CommitAll()

	rebased, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, uint64(0), rebased.StartBlock, "the window must restart at zero")
	require.Equal(t, remaining, rebased.EndBlock,
		"the agreement must carry only the time it had left, not its old absolute end")
}

// TestZeroHeightRebase_MovesTheExpirationIndex is the half that actually makes
// the agreement expire. Rebasing the record without moving the index would leave
// it indexed at the old absolute height, which the new chain never reaches.
func TestZeroHeightRebase_MovesTheExpirationIndex(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)
	advanceBlocks(f, 20)

	stored, _ := f.k.GetAgreement(f.ctx, agreement.Id)
	oldEnd := stored.EndBlock
	require.Contains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, oldEnd), agreement.Id)

	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	require.NoError(t, cc.RebaseAgreementsForZeroHeightGenesis())
	cc.CommitAll()

	require.NotContains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, oldEnd), agreement.Id,
		"the row at the old absolute height must be cleared")
	require.Contains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, 30), agreement.Id,
		"the agreement must be indexed at its rebased end block")
}

/* TestZeroHeightRebase_ExhaustedAgreementGetsOneBlock covers an agreement with
 * nothing left. Zero would index it at a height the chain never reaches, and an
 * expiry gets exactly one attempt - so it would hold capacity in the provider's
 * load forever rather than settling.
 */
func TestZeroHeightRebase_ExhaustedAgreementGetsOneBlock(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	agreement, _ := f.openAgreement(t, 100, 50)
	advanceBlocks(f, 50) // sitting exactly on the end block

	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	require.NoError(t, cc.RebaseAgreementsForZeroHeightGenesis())
	cc.CommitAll()

	rebased, found := f.k.GetAgreement(f.ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, uint64(1), rebased.EndBlock,
		"an exhausted agreement must expire on the new chain's first block, not never")
	require.Contains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, 1), agreement.Id)
}

/* TestZeroHeightRebase_ProviderCheckpointFollows pins the two clocks moving
 * together. The provider is settled against the old clock first - so the span it
 * actually served is paid - and only then rebased.
 */
func TestZeroHeightRebase_ProviderCheckpointFollows(t *testing.T) {
	f := setupTeardownFixture(t, 10, "0.5", "0.25")

	f.openAgreement(t, 100, 50)
	advanceBlocks(f, 20)

	cc := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx))
	require.NoError(t, cc.RebaseAgreementsForZeroHeightGenesis())
	cc.CommitAll()

	provider := f.k.NewCurrentContext(sdk.UnwrapSDKContext(f.ctx)).GetProvider(f.provider.Id)
	require.Equal(t, uint64(0), provider.GetCheckpointBlock(),
		"the checkpoint clock must restart with the windows it bills against")
}
