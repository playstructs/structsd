// Command addressproof mints a fresh keypair and prints an address registration
// proof for it.
//
// tests/test_chain.sh used to carry a hardcoded address, pubkey and signature.
// That worked only for as long as the signed payload never changed, and the
// payload is exactly the thing that has to change when replay protection is
// added — a stale literal there does not fail loudly, it fails as one phase
// quietly not testing anything. Building the proof here means the script signs
// with the same types.AddressRegisterProofInput the chain verifies with, so the
// two cannot drift.
//
// Usage:
//
//	go run ./tests/tools/addressproof -chain-id structstestnet-00 -player 1-1 [-nonce 0]
//
// Prints three shell-friendly lines: ADDRESS, PROOF_PUBKEY, PROOF_SIGNATURE.
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	"structs/x/structs/types"
)

func main() {
	chainId := flag.String("chain-id", "", "chain id the proof is bound to")
	playerId := flag.String("player", "", "player id the address is joining")
	nonce := flag.Uint64("nonce", 0, "current registration-proof nonce for the address")
	flag.Parse()

	if *chainId == "" || *playerId == "" {
		fmt.Fprintln(os.Stderr, "addressproof: -chain-id and -player are required")
		os.Exit(1)
	}

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()

	// The handler rebuilds the address from the pubkey and requires a match, so
	// this must be the module's own derivation rather than sdk.AccAddress.
	address := types.PubKeyToBech32(pubKey.Bytes())

	signature, err := privKey.Sign([]byte(types.AddressRegisterProofInput(*chainId, *playerId, address, *nonce)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "addressproof: signing failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ADDRESS=%s\n", address)
	fmt.Printf("PROOF_PUBKEY=%s\n", hex.EncodeToString(pubKey.Bytes()))
	fmt.Printf("PROOF_SIGNATURE=%s\n", hex.EncodeToString(signature))
}
