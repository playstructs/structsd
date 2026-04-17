package ante_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

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

func TestPubKeyDerivation_NonSignatureMessageSkipped(t *testing.T) {
	dec := ante.NewPubKeyDerivationDecorator()
	next, called := identityHandler()

	msg := &types.MsgFleetMove{Creator: "structs1alice", FleetId: "2-1", DestinationLocationId: "7-1"}
	tx := mockTx{msgs: []sdk.Msg{msg}}

	_, err := dec.AnteHandle(newTestCtx(), tx, false, next)
	require.NoError(t, err)
	require.True(t, *called)
}
