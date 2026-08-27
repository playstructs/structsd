package app_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"structs/app"
	structstypes "structs/x/structs/types"
)

/* Regression suite for a zero-height export that killed the exporter.
 *
 * The jail allowlist marks every validator outside it as jailed. SetValidator
 * writes only the validator record - the power index is a separate structure
 * with its own setter and deleter - so flagging Jailed left a jailed validator
 * in the power store, and ApplyAndReturnValidatorSetUpdates walks that store and
 * refuses: "should never retrieve a jailed validator from the power store". The
 * error went to log.Fatal, which exits without unwinding, so excluding any
 * active validator inside MaxValidators produced no genesis at all.
 *
 * That is the worst time for it: a planned migration, with the chain already
 * stopped, and nothing to show for the export but an exit code.
 */

// newBondedValidator creates and bonds a validator through the real staking
// message server, so it lands in the power index the way a live one does.
func newBondedValidator(t *testing.T, bApp *app.App, ctx sdk.Context, moniker string) sdk.ValAddress {
	t.Helper()

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	operatorAcc := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	stake := math.NewInt(10_000_000)
	funding := sdk.NewCoins(sdk.NewCoin(bondDenom, stake.MulRaw(2)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, funding))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, operatorAcc, funding))

	valAddr := sdk.ValAddress(operatorAcc)
	createMsg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		ed25519.GenPrivKey().PubKey(),
		sdk.NewCoin(bondDenom, stake),
		stakingtypes.NewDescription(moniker, "", "", "", ""),
		stakingtypes.NewCommissionRates(
			math.LegacyNewDecWithPrec(1, 1),
			math.LegacyNewDecWithPrec(2, 1),
			math.LegacyNewDecWithPrec(1, 2),
		),
		math.OneInt(),
	)
	require.NoError(t, err)

	_, err = stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper).CreateValidator(ctx, createMsg)
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	validator, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.True(t, validator.IsBonded(), "fixture sanity: the validator must be in the power index")

	return valAddr
}

/* TestJailingRequiresPowerIndexRemoval pins the SDK behaviour the fix rests on.
 *
 * SetValidator writes only the validator record; the power index is a separate
 * structure with its own setter and deleter. So flagging Jailed and storing the
 * validator leaves a jailed validator in the power store, and
 * ApplyAndReturnValidatorSetUpdates - which walks that store - refuses outright.
 *
 * The first half reproduces what prepForZeroHeightGenesis used to do and asserts
 * the refusal, so this test fails if a future SDK stops caring and the fix
 * silently becomes unnecessary. The second half is the fix.
 */
func TestJailingRequiresPowerIndexRemoval(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	valAddr := newBondedValidator(t, bApp, ctx, "excluded")

	validator, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)

	// The old behaviour: flag and store, leaving the power index alone.
	validator.Jailed = true
	require.NoError(t, bApp.StakingKeeper.SetValidator(ctx, validator))

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Error(t, err,
		"a jailed validator left in the power index must break the validator set update; that is what killed the export")
	require.Contains(t, err.Error(), "jailed validator")

	// The fix: take it out of the power index first.
	require.NoError(t, bApp.StakingKeeper.DeleteValidatorByPowerIndex(ctx, validator))

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err, "removing the power-index entry must let the update through")

	jailed, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.True(t, jailed.IsJailed())
}

/* TestZeroHeightExportRejectsMalformedAllowlistAddress covers the other half:
 * an operator's typo must come back as an error rather than an exit code.
 *
 * This reaches prepForZeroHeightGenesis and returns from it before any module
 * export, which is why it can assert on the returned error - the surrounding
 * export needs IBC transfer state this fixture does not build.
 */
func TestZeroHeightExportRejectsMalformedAllowlistAddress(t *testing.T) {
	bApp, _ := setupJailGateApp(t)

	_, err := bApp.ExportAppStateAndValidators(true, []string{"not-a-validator-address"}, nil)
	require.Error(t, err, "a malformed allowlist entry must be reported, not fatal")
	require.Contains(t, err.Error(), "allowlist")
}
