package keeper

import (
	"context"

	//"github.com/cosmos/cosmos-sdk/runtime"
	//"cosmossdk.io/store/prefix"
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	//"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"

    "encoding/hex"
    crypto "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
    //"fmt"
    //"encoding/binary"
    //"strings"
    //"strconv"
)

/*
    message QueryValidateSignatureRequest {
      string address        = 1;
      string message        = 2;
      string proofPubKey    = 3;
      string proofSignature = 4;
    }

    message QueryValidateSignatureResponse {
      bool pubkeyFormatError      = 1;
      bool signatureFormatError   = 2;
      bool addressPubkeyMismatch  = 3;
      bool signatureInvalid       = 4;
      bool valid                  = 5;
    }
*/

func (k Keeper) ValidateSignature(goCtx context.Context, req *types.QueryValidateSignatureRequest) (*types.QueryValidateSignatureResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    response := types.QueryValidateSignatureResponse{}
    var invalid bool

    decodedProofPubKey, decodeErr := hex.DecodeString(req.ProofPubKey)
    if decodeErr != nil {
        response.PubkeyFormatError = true
        response.SignatureInvalid = true
        invalid = true
    }

    // Convert provided pub key into a bech32 string (i.e., an address)
	address := types.PubKeyToBech32(decodedProofPubKey)

    if (address != req.Address) {
        response.AddressPubkeyMismatch = true
        response.SignatureInvalid = true
        invalid = true
    }

    pubKey := crypto.PubKey{}
    pubKey.Key = decodedProofPubKey

    // Decode the Signature from Hex Encoding
    decodedProofSignature, decodeErr := hex.DecodeString(req.ProofSignature)
    if decodeErr != nil {
        response.SignatureFormatError = true
        response.SignatureInvalid = true
        invalid = true
    }

    // Proof needs to only be 64 characters. Some systems provide a checksum bit on the end that ruins it all
    if !invalid {
        if pubKey.VerifySignature([]byte(req.Message), decodedProofSignature[:64]) {
            response.Valid = true
        } else {
            response.SignatureInvalid = true
        }
    }

	return &response, nil
}
