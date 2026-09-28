package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Pins an SDK integration constraint, not desired vault behavior. Unbond reads
// a validator before delegation hooks, then passes that captured value to
// RemoveValidatorTokensAndShares afterwards. A hook-only minimum-self-stake jail
// would be overwritten. This isolates that final write without installing a new
// hook chain or pretending the self-delegation migration is implemented.
func TestSDKCapturedValidatorWriteCanUndoHookJail(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	captured := validators[0]
	require.False(t, captured.Jailed)
	operator, err := sdk.ValAddressFromBech32(captured.OperatorAddress)
	require.NoError(t, err)
	consensus, err := captured.GetConsAddr()
	require.NoError(t, err)
	require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
	jailed, err := bApp.StakingKeeper.GetValidator(ctx, operator)
	require.NoError(t, err)
	require.True(t, jailed.Jailed)
	// Zero shares isolate the overwrite from token movement; this uses the
	// same write that Unbond invokes with a positive removal after its hooks.
	_, removed, err := bApp.StakingKeeper.RemoveValidatorTokensAndShares(ctx, captured, math.LegacyZeroDec())
	require.NoError(t, err)
	require.True(t, removed.IsZero())
	after, err := bApp.StakingKeeper.GetValidator(ctx, operator)
	require.NoError(t, err)
	require.False(t, after.Jailed, "a late hook jail does not survive the SDK's captured validator write")
	require.Equal(t, captured.Tokens, after.Tokens)
	require.Equal(t, captured.DelegatorShares, after.DelegatorShares)
}
