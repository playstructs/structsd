package keeper

import (
	"context"
	"fmt"

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

    // Is the calling account a player?
    playerId := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerId > 0) {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Non-player account cannot associate new addresses with themselves")
    }

	// Is the address associated with an account yet
    playerFoundForAddress := k.GetPlayerIndexFromAddress(ctx, msg.Address)
    if (playerFoundForAddress > 0) {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not associate an address when already has an account")
    }


	// Does this creator address have the permissions to do this
    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssociations)) {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionAssociation, "Calling address (%s) has no Association permissions ", msg.Creator)
    }
    // The calling address must have a minimum of the same permission level
    if (!k.PermissionHasAll(ctx, addressPermissionId, types.Permission(msg.Permissions))) {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionAssociation, "Calling address (%s) does not have permissions needed to allow address association of higher functionality ", msg.Creator)
    }


	// Does the signature verify in the proof
    // Convert provided pub key into a bech32 string (i.e., an address)
	address := types.PubKeyToBech32(msg.ProofPubKey)
    if (address != msg.Address) {
         return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Proof mismatch for %s vs %s vs %s", address, msg.Address)
    }

    pubKey := crypto.PubKey{}
    pubKey.Key = msg.ProofPubKey

    // We rebuild the message manually here rather than trust the client to provide it
    hashInput := fmt.Sprintf("PLAYER%dADDRESS%s", playerId, msg.Address)


    // Proof needs to only be 64 characters. Some systems provide a checksum bit on the end that ruins it all
    if (!pubKey.VerifySignature([]byte(hashInput), msg.ProofSignature[:64])) {
         return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Proof signature verification failure")
    }

	// Add the address and player index to the keeper
    k.SetPlayerIndexForAddress(ctx, msg.Address, playerId)

	// Add the permission to the new address
    newAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    k.PermissionAdd(ctx, newAddressPermissionId, types.Permission(msg.Permissions))

	return &types.MsgAddressRegisterResponse{}, nil
}
