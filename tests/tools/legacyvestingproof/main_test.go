package main

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestLegacyVestingProofAndKeyringAgreeWithoutExportingPrivateKey(t *testing.T) {
	home := t.TempDir()
	proof, err := provision(home, "upgrade_vesting_delayed", "1-7")
	require.NoError(t, err)
	publicBytes, err := hex.DecodeString(proof.PublicKey)
	require.NoError(t, err)
	public := &secp256k1.PubKey{Key: publicBytes}
	signature, err := hex.DecodeString(proof.Signature)
	require.NoError(t, err)
	require.True(t, public.VerifySignature([]byte("PLAYER1-7ADDRESS"+proof.Address), signature))
	require.False(t, public.VerifySignature([]byte("PLAYER1-8ADDRESS"+proof.Address), signature))
	require.False(t, public.VerifySignature([]byte(types.AddressRegisterProofInput("new-chain", "1-7", proof.Address, 0)), signature))
	require.Equal(t, types.PubKeyToBech32(publicBytes), proof.Address)
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	keys, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, home, strings.NewReader(""), codec.NewProtoCodec(registry))
	require.NoError(t, err)
	stored, err := keys.Key(proof.KeyName)
	require.NoError(t, err)
	storedPublic, err := stored.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, publicBytes, storedPublic.Bytes())
	_, err = provision(home, proof.KeyName, "1-7")
	require.Error(t, err, "a retry must not silently overwrite an existing fixture identity")
	unchanged, err := keys.Key(proof.KeyName)
	require.NoError(t, err)
	unchangedPublic, err := unchanged.GetPubKey()
	require.NoError(t, err)
	require.Equal(t, publicBytes, unchangedPublic.Bytes())
	encoded, err := json.Marshal(proof)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(encoded, &fields))
	require.Len(t, fields, 5)
	for _, name := range []string{"key_name", "player_id", "address", "public_key", "signature"} {
		require.Contains(t, fields, name)
	}
}

func TestLegacyVestingProofRefusesNonFixtureNamesAndInvalidPlayers(t *testing.T) {
	for _, player := range []string{"", "1-0", "1-01", "0-1", "1-18446744073709551616"} {
		_, err := provision(t.TempDir(), "upgrade_vesting_delayed", player)
		require.Error(t, err)
	}
	for _, name := range []string{"alice", "upgrade_vesting_", "../upgrade_vesting_delayed"} {
		_, err := provision(t.TempDir(), name, "1-7")
		require.Error(t, err)
	}
	_, err := provision(t.TempDir()+"/missing", "upgrade_vesting_delayed", "1-7")
	require.Error(t, err)
}
