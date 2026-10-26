// legacyvestingproof provisions a key in an isolated rehearsal's test keyring
// and emits only the public v0.21 registration proof. It deliberately signs the
// old release's payload; it must not be used for v0.22 registration.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

type registrationProof struct {
	KeyName   string `json:"key_name"`
	PlayerID  string `json:"player_id"`
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

func provision(home, name, player string) (registrationProof, error) {
	var proof registrationProof
	kind, _, err := types.ParseAssetOwnerID(player)
	if err != nil || kind != types.ObjectType_player {
		return proof, fmt.Errorf("legacy vesting proof requires a canonical player id")
	}
	if !regexp.MustCompile(`^upgrade_vesting_[a-z0-9_]+$`).MatchString(name) {
		return proof, fmt.Errorf("key name must use the upgrade_vesting_ fixture prefix")
	}
	info, err := os.Stat(home)
	if err != nil || !info.IsDir() {
		return proof, fmt.Errorf("an existing isolated rehearsal home is required")
	}
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	keys, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, home, strings.NewReader(""), codec.NewProtoCodec(registry))
	if err != nil {
		return proof, err
	}
	private := secp256k1.GenPrivKey()
	public := private.PubKey()
	address := types.PubKeyToBech32(public.Bytes())
	// This literal is the payload rebuilt by v0.21.0 AddressRegister. Keeping
	// it here prevents an accidental dependency on the current proof format.
	signature, err := private.Sign([]byte("PLAYER" + player + "ADDRESS" + address))
	if err != nil {
		return proof, err
	}
	// Import refuses an existing name. Private bytes never enter stdout, a
	// subprocess argument, or the fixture's public evidence files.
	if err := keys.ImportPrivKeyHex(name, hex.EncodeToString(private.Bytes()), "secp256k1"); err != nil {
		return proof, err
	}
	return registrationProof{name, player, address, hex.EncodeToString(public.Bytes()), hex.EncodeToString(signature)}, nil
}

func main() {
	home := flag.String("home", "", "isolated rehearsal home (test keyring only)")
	name := flag.String("name", "", "new upgrade_vesting_ key name; existing names are refused")
	player := flag.String("player", "", "v0.21 player receiving the address association")
	flag.Parse()
	proof, err := provision(*home, *name, *player)
	if err != nil {
		fmt.Fprintln(os.Stderr, "legacyvestingproof:", err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(proof); err != nil {
		fmt.Fprintln(os.Stderr, "legacyvestingproof:", err)
		os.Exit(1)
	}
}
