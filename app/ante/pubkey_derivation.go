package ante

import (
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// PubKeyDerivationDecorator validates that the proofPubKey in signature-bearing
// Structs messages derives to the claimed address. This is a cheap pre-filter
// (no state reads) that catches garbage or mismatched pubkeys before the
// handler runs the full secp256k1 verification.
type PubKeyDerivationDecorator struct{}

func NewPubKeyDerivationDecorator() PubKeyDerivationDecorator {
	return PubKeyDerivationDecorator{}
}

type pubKeyMessage interface {
	GetProofPubKey() string
	GetAddress() string
}

func (d PubKeyDerivationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	for _, msg := range tx.GetMsgs() {
		typeURL := sdk.MsgTypeURL(msg)
		if !SignatureMessages[typeURL] {
			continue
		}

		pkMsg, ok := msg.(pubKeyMessage)
		if !ok {
			continue
		}

		pubKeyHex := pkMsg.GetProofPubKey()
		claimedAddr := pkMsg.GetAddress()

		if pubKeyHex == "" || claimedAddr == "" {
			return ctx, fmt.Errorf("structs ante: %s missing proofPubKey or address", typeURL)
		}

		decoded, err := hex.DecodeString(pubKeyHex)
		if err != nil {
			return ctx, fmt.Errorf("structs ante: %s invalid proofPubKey hex: %w", typeURL, err)
		}

		if len(decoded) != 33 {
			return ctx, fmt.Errorf("structs ante: %s proofPubKey must be 33 bytes (compressed secp256k1), got %d", typeURL, len(decoded))
		}

		derivedAddr := types.PubKeyToBech32(decoded)
		if derivedAddr != claimedAddr {
			return ctx, fmt.Errorf("structs ante: %s proofPubKey derives to %s, expected %s", typeURL, derivedAddr, claimedAddr)
		}
	}

	return next(ctx, tx, simulate)
}
