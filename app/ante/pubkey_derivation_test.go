package ante_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/app/ante"
	"structs/x/structs/types"
)

func TestPubKeyDerivation_ValidPubKey(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, called := identityHandler()

	privKey := secp256k1.GenPrivKey()
	pubKeyBytes := privKey.PubKey().Bytes()
	pubKeyHex := hex.EncodeToString(pubKeyBytes)
	derivedAddr := types.PubKeyToBech32(pubKeyBytes)

	msg := &types.MsgAddressRegister{
		Creator:        "structs1someone",
		Address:        derivedAddr,
		ProofPubKey:    pubKeyHex,
		ProofSignature: "deadbeef",
		PlayerId:       "1-1",
	}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestPubKeyDerivation_MismatchedAddress(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, _ := identityHandler()

	privKey := secp256k1.GenPrivKey()
	pubKeyHex := hex.EncodeToString(privKey.PubKey().Bytes())

	msg := &types.MsgAddressRegister{
		Creator:        "structs1someone",
		Address:        "structs1wrongaddress",
		ProofPubKey:    pubKeyHex,
		ProofSignature: "deadbeef",
		PlayerId:       "1-1",
	}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.True(t, ante.ErrInvalidProofIdentity.Is(err))
	require.Contains(t, err.Error(), "proofPubKey derives to")
}

func TestPubKeyDerivation_InvalidHex(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, _ := identityHandler()

	msg := &types.MsgAddressRegister{
		Creator:        "structs1someone",
		Address:        "structs1test",
		ProofPubKey:    "not-valid-hex!!",
		ProofSignature: "deadbeef",
		PlayerId:       "1-1",
	}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid proofPubKey hex")
}

func TestPubKeyDerivation_WrongLength(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, _ := identityHandler()

	shortKey := hex.EncodeToString(make([]byte, 20))
	msg := &types.MsgAddressRegister{
		Creator:        "structs1someone",
		Address:        "structs1test",
		ProofPubKey:    shortKey,
		ProofSignature: "deadbeef",
		PlayerId:       "1-1",
	}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be 33 bytes")
}

func TestPubKeyDerivation_EmptyPubKey(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, _ := identityHandler()

	msg := &types.MsgAddressRegister{
		Creator:        "structs1someone",
		Address:        "structs1test",
		ProofPubKey:    "",
		ProofSignature: "deadbeef",
		PlayerId:       "1-1",
	}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing proofPubKey or address")
}

func TestPubKeyDerivation_EmptyAddress(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, _ := identityHandler()

	privKey := secp256k1.GenPrivKey()
	pubKeyHex := hex.EncodeToString(privKey.PubKey().Bytes())

	msg := &types.MsgAddressRegister{
		Creator:        "structs1someone",
		Address:        "",
		ProofPubKey:    pubKeyHex,
		ProofSignature: "deadbeef",
		PlayerId:       "1-1",
	}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing proofPubKey or address")
}

/* A guild charter's consent signature is optional: the solo case carries none,
 * because a founder who signs the transaction has already proved who they are.
 * The decorator therefore has to tell "no consent" apart from "malformed
 * consent", and only the first may skip.
 *
 * Half-populated is the case worth having a test for. If it skipped too, the
 * pre-filter would be bypassable by simply omitting one of the two fields, and
 * the handler's own derivation would silently become the only guard.
 */
func TestPubKeyDerivation_OptionalConsentAbsentIsSkipped(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, called := identityHandler()

	msg := &types.MsgGuildCreate{
		Creator:   "structs1solver",
		ReactorId: "3-1",
		Endpoint:  "endpoint",
		Proof:     "deadbeef",
		Nonce:     "1",
	}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}

func TestPubKeyDerivation_OptionalConsentPresentIsChecked(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, called := identityHandler()

	privKey := secp256k1.GenPrivKey()
	pubKeyBytes := privKey.PubKey().Bytes()

	msg := &types.MsgGuildCreate{
		Creator:         "structs1solver",
		ReactorId:       "3-1",
		Endpoint:        "endpoint",
		FounderPlayerId: "1-2",
		Address:         types.PubKeyToBech32(pubKeyBytes),
		ProofPubKey:     hex.EncodeToString(pubKeyBytes),
		ProofSignature:  "deadbeef",
	}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)

	msg.Address = "structs1wrongaddress"
	_, err = dec.AnteHandle(newTestCtx(), mockTx{msgs: []sdk.Msg{msg}}, false, next)
	require.Error(t, err)
	require.Contains(t, err.Error(), "proofPubKey derives to")
}

func TestPubKeyDerivation_OptionalConsentHalfPopulatedRejected(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, _ := identityHandler()

	privKey := secp256k1.GenPrivKey()
	pubKeyHex := hex.EncodeToString(privKey.PubKey().Bytes())

	for _, tc := range []struct {
		name        string
		address     string
		proofPubKey string
	}{
		{"pubkey without address", "", pubKeyHex},
		{"address without pubkey", "structs1founder", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := &types.MsgGuildCreate{
				Creator:     "structs1solver",
				ReactorId:   "3-1",
				Address:     tc.address,
				ProofPubKey: tc.proofPubKey,
			}
			_, err := dec.AnteHandle(newTestCtx(), mockTx{msgs: []sdk.Msg{msg}}, false, next)
			require.Error(t, err)
			require.Contains(t, err.Error(), "missing proofPubKey or address")
		})
	}
}

func TestPubKeyDerivation_NonSignatureMessageSkipped(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}
