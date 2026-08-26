package app_test

import (
	"encoding/hex"
	"errors"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	structskeeper "structs/x/structs/keeper"
	structstypes "structs/x/structs/types"
)

/* TestAddressRegisterProofCannotBeReplayedAfterRevoke is the regression on a
 * registration proof that was a permanent bearer credential rather than a
 * one-time consent.
 *
 * The signed payload used to be "PLAYER<id>ADDRESS<addr>" — a pure function of
 * two values that never change. The only thing stopping a second use was the
 * address index, and AddressRevoke deletes exactly that. So the player side
 * could revoke an address, wait for it to receive funds again, resubmit the
 * original proof, and sweep the balance a second time with no fresh consent
 * from the key that owns it. No third party had to cooperate: PermissionCheck
 * passes an owner acting on their own player object.
 *
 * The nonce is what closes it, and the assertion that matters most here is the
 * one that looks like a no-op: the nonce row must still be there after the
 * revoke. Clearing it alongside the association would hand the proof back its
 * validity at exactly the moment the address stops being watched.
 */
func TestAddressRegisterProofCannotBeReplayedAfterRevoke(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	primaryAcc := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	player := cc.UpsertPlayer(primaryAcc.String())
	playerId := player.GetPlayerId()
	cc.CommitAll()

	// The registering key has to be secp256k1: the handler rebuilds the address
	// from the supplied pubkey and requires it to match.
	privKey := secp256k1.GenPrivKey()
	secondaryAcc := sdk.AccAddress(privKey.PubKey().Address())
	require.Equal(t, secondaryAcc.String(), structstypes.PubKeyToBech32(privKey.PubKey().Bytes()))

	require.Zero(t, bApp.StructsKeeper.GetAddressProofNonce(ctx, secondaryAcc.String()),
		"an address that has never registered starts at nonce 0")

	proofInput := structstypes.AddressRegisterProofInput(
		ctx.ChainID(), playerId, secondaryAcc.String(), 0)
	signature, err := privKey.Sign([]byte(proofInput))
	require.NoError(t, err)

	msg := &structstypes.MsgAddressRegister{
		Creator:        primaryAcc.String(),
		PlayerId:       playerId,
		Address:        secondaryAcc.String(),
		Permissions:    uint64(structstypes.PermAll),
		ProofPubKey:    hex.EncodeToString(privKey.PubKey().Bytes()),
		ProofSignature: hex.EncodeToString(signature),
	}

	firstBalance := math.NewInt(3_000_000)
	fundRedelegationAccount(t, bApp, ctx, secondaryAcc, sdk.NewCoins(sdk.NewCoin(bondDenom, firstBalance)))

	structsMsgServer := structskeeper.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = structsMsgServer.AddressRegister(ctx, msg)
	require.NoError(t, err, "a proof signed against the current nonce registers")

	require.Equal(t, firstBalance, bApp.BankKeeper.GetBalance(ctx, primaryAcc, bondDenom).Amount,
		"registration sweeps the address balance to the primary")
	require.Equal(t, uint64(1), bApp.StructsKeeper.GetAddressProofNonce(ctx, secondaryAcc.String()),
		"registering burns the nonce the proof was signed against")

	_, err = structsMsgServer.AddressRevoke(ctx, &structstypes.MsgAddressRevoke{
		Creator: primaryAcc.String(),
		Address: secondaryAcc.String(),
	})
	require.NoError(t, err)

	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, secondaryAcc.String()),
		"the association is gone")
	require.Equal(t, uint64(1), bApp.StructsKeeper.GetAddressProofNonce(ctx, secondaryAcc.String()),
		"the nonce must outlive the association it protected, or the proof goes live again")

	// The address earns again after leaving. This is the balance the replay was
	// after, and it has to still be here at the end.
	secondBalance := math.NewInt(5_000_000)
	fundRedelegationAccount(t, bApp, ctx, secondaryAcc, sdk.NewCoins(sdk.NewCoin(bondDenom, secondBalance)))

	primaryBefore := bApp.BankKeeper.GetBalance(ctx, primaryAcc, bondDenom).Amount

	_, err = structsMsgServer.AddressRegister(ctx, msg)
	require.Error(t, err, "the original proof must not register the address a second time")
	requireSignatureInvalid(t, err)

	require.Equal(t, secondBalance, bApp.BankKeeper.GetBalance(ctx, secondaryAcc, bondDenom).Amount,
		"the replay must not have swept anything")
	require.Equal(t, primaryBefore, bApp.BankKeeper.GetBalance(ctx, primaryAcc, bondDenom).Amount)
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, secondaryAcc.String()),
		"and must not have re-associated the address")

	// A fresh proof against the burned nonce's successor still works, so the
	// nonce closes replay without stranding an address that wants back in.
	reproofInput := structstypes.AddressRegisterProofInput(
		ctx.ChainID(), playerId, secondaryAcc.String(), 1)
	resignature, err := privKey.Sign([]byte(reproofInput))
	require.NoError(t, err)

	msg.ProofSignature = hex.EncodeToString(resignature)
	_, err = structsMsgServer.AddressRegister(ctx, msg)
	require.NoError(t, err, "re-registering with a freshly signed proof is still allowed")

	require.Equal(t, uint64(2), bApp.StructsKeeper.GetAddressProofNonce(ctx, secondaryAcc.String()))
	require.Equal(t, primaryBefore.Add(secondBalance),
		bApp.BankKeeper.GetBalance(ctx, primaryAcc, bondDenom).Amount)
}

// A proof carries no meaning off the chain it was signed for. Player ids are
// small integers, so the same id exists on every Structs chain.
func TestAddressRegisterProofIsBoundToTheChain(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	primaryAcc := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	player := cc.UpsertPlayer(primaryAcc.String())
	playerId := player.GetPlayerId()
	cc.CommitAll()

	privKey := secp256k1.GenPrivKey()
	secondaryAcc := sdk.AccAddress(privKey.PubKey().Address())

	foreign := structstypes.AddressRegisterProofInput(
		"some-other-structs-chain", playerId, secondaryAcc.String(), 0)
	require.NotEqual(t, foreign,
		structstypes.AddressRegisterProofInput(ctx.ChainID(), playerId, secondaryAcc.String(), 0))

	signature, err := privKey.Sign([]byte(foreign))
	require.NoError(t, err)

	structsMsgServer := structskeeper.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = structsMsgServer.AddressRegister(ctx, &structstypes.MsgAddressRegister{
		Creator:        primaryAcc.String(),
		PlayerId:       playerId,
		Address:        secondaryAcc.String(),
		Permissions:    uint64(structstypes.PermAll),
		ProofPubKey:    hex.EncodeToString(privKey.PubKey().Bytes()),
		ProofSignature: hex.EncodeToString(signature),
	})
	require.Error(t, err, "a proof signed for another chain must not register here")
	requireSignatureInvalid(t, err)
}

// requireSignatureInvalid asserts the rejection was the proof failing to verify,
// rather than any of the other reasons AddressRegister can refuse.
func requireSignatureInvalid(t *testing.T, err error) {
	t.Helper()

	var validationErr *structstypes.AddressValidationError
	require.True(t, errors.As(err, &validationErr), "expected an AddressValidationError, got %v", err)
	require.Equal(t, "signature_invalid", validationErr.Reason)
}
