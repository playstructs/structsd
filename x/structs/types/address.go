package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/btcutil/bech32"
    "crypto/sha256"
    "golang.org/x/crypto/ripemd160"
)

// Thank you Sunny
// https://gist.github.com/arnabghose997/f2d33955d5359de255055d6ce8c619f7

func PubKeyToBech32(pubKey []byte) string {

    // Hash pubKeyBytes as: RIPEMD160(SHA256(public_key_bytes))
	pubKeySha256Hash := sha256.Sum256(pubKey)
	ripemd160hash := ripemd160.New()
	ripemd160hash.Write(pubKeySha256Hash[:])
	addressBytes := ripemd160hash.Sum(nil)

	return ToBech32("structs", addressBytes)
}

// Code courtesy: https://github.com/cosmos/cosmos-sdk/blob/90c9c9a9eb4676d05d3f4b89d9a907bd3db8194f/types/bech32/bech32.go#L10
func ToBech32(addrPrefix string, addrBytes []byte) string {
  converted, err := bech32.ConvertBits(addrBytes, 8, 5, true)
	if err != nil {
		panic(err)
	}

	addr, err := bech32.Encode(addrPrefix, converted)
  if err != nil {
    panic(err)
  }

  return addr
}