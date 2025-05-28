package keeper

import (
	"context"
	"fmt"
    "encoding/hex"
    "math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

    crypto "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
)

func (k msgServer) AddressRegister(goCtx context.Context, msg *types.MsgAddressRegister) (*types.MsgAddressRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


    player, err := k.GetPlayerCacheFromId(ctx, msg.PlayerId)
    if err != nil {
       return &types.MsgAddressRegisterResponse{}, err
    }

    if !player.LoadPlayer() {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Non-player account cannot associate new addresses with themselves")
    }


	// Is the address associated with an account yet
    playerFoundForAddress := k.GetPlayerIndexFromAddress(ctx, msg.Address)
    if (playerFoundForAddress > 0) {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not associate an address when already has an account")
    }

     // Check if msg.Creator has PermissionAssociations on the Address and Account
    err = player.CanBeAdministratedBy(msg.Creator, types.PermissionAssociations)
    if err != nil {
       return &types.MsgAddressRegisterResponse{}, err
    }

	// Does this creator address have the permissions to do this
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // The calling address must have a minimum of the same permission level
    if (!k.PermissionHasAll(ctx, addressPermissionId, types.Permission(msg.Permissions))) {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionAssociation, "Calling address (%s) does not have permissions needed to allow address association of higher functionality ", msg.Creator)
    }


	// Does the signature verify in the proof
	// Decode the PubKey from hex Encoding
    fmt.Println("Encoding string:", msg.ProofPubKey)

    decodedProofPubKey, decodeErr := hex.DecodeString(msg.ProofPubKey)
    if decodeErr != nil {
        fmt.Println("Error decoding string:", decodeErr)
    }
    // Convert provided pub key into a bech32 string (i.e., an address)
	address := types.PubKeyToBech32(decodedProofPubKey)
    if (address != msg.Address) {
         return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "Proof mismatch for %s vs %s", address, msg.Address)
    }

    pubKey := crypto.PubKey{}
    pubKey.Key = decodedProofPubKey

    // We rebuild the message manually here rather than trust the client to provide it
    hashInput := fmt.Sprintf("PLAYER%sADDRESS%s", msg.PlayerId, msg.Address)
    fmt.Println(hashInput)

    // Decode the Signature from Hex Encoding
    decodedProofSignature, decodeErr := hex.DecodeString(msg.ProofSignature)
    if decodeErr != nil {
        fmt.Println("Error decoding string:", decodeErr)
    }

    // Proof needs to only be 64 characters. Some systems provide a checksum bit on the end that ruins it all
    if (!pubKey.VerifySignature([]byte(hashInput), decodedProofSignature[:64])) {
         return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPlayerUpdate, "Proof signature verification failure")
    }

	// Add the address and player index to the keeper
    k.SetPlayerIndexForAddress(ctx, msg.Address, player.GetIndex())

	// Add the permission to the new address
    newAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    k.PermissionAdd(ctx, newAddressPermissionId, types.Permission(msg.Permissions))


    // Move Funds
    primaryAcc, _   := sdk.AccAddressFromBech32(player.GetPrimaryAddress())
    newAcc, _   := sdk.AccAddressFromBech32(msg.Address)

    // Get Balance
    balances := k.bankKeeper.SpendableCoins(ctx, newAcc)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, newAcc, primaryAcc, balances)
    if err != nil {
        return &types.MsgAddressRegisterResponse{}, err
    }

    // Move Reactor Infusions over
    primaryDelegations, _ := k.stakingKeeper.GetDelegatorDelegations(ctx, newAcc, math.MaxUint16)
    for _, delegation := range primaryDelegations {
        k.stakingKeeper.RemoveDelegation(ctx, delegation)

        delegation.DelegatorAddress = player.GetPrimaryAddress()
        k.stakingKeeper.SetDelegation(ctx, delegation)
    }


	return &types.MsgAddressRegisterResponse{}, nil
}
