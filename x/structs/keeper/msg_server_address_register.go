package keeper

import (
	"context"
    "encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

    crypto "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
)

func (k msgServer) AddressRegister(goCtx context.Context, msg *types.MsgAddressRegister) (*types.MsgAddressRegisterResponse, error) {
    emptyResponse := &types.MsgAddressRegisterResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }


    player, err := cc.GetExistingPlayer(msg.PlayerId)
    if err != nil {
       return emptyResponse, err
    }

	// Is the address associated with an account yet
    playerFoundForAddress := cc.GetPlayerIndexFromAddress(msg.Address)
    if (playerFoundForAddress > 0) {
        return emptyResponse, types.NewAddressValidationError(msg.Address, "already_registered")
    }

    err = player.CanRegisterAddressBy(activePlayer, types.Permission(msg.Permissions));
    if err != nil {
       return emptyResponse, err
    }

	// Does the signature verify in the proof
	// Decode the PubKey from hex Encoding
    k.logger.Info("Address Register", "encodingString", msg.ProofPubKey)

    decodedProofPubKey, decodeErr := hex.DecodeString(msg.ProofPubKey)
    if decodeErr != nil {
        k.logger.Error("Address Register Public Key", "decodingError", decodeErr)
        return emptyResponse, decodeErr
    }
    // Convert provided pub key into a bech32 string (i.e., an address)
	address := types.PubKeyToBech32(decodedProofPubKey)
    if (address != msg.Address) {
         return emptyResponse, types.NewAddressValidationError(msg.Address, "proof_mismatch").WithPlayers(address, msg.Address)
    }

    pubKey := crypto.PubKey{}
    pubKey.Key = decodedProofPubKey

    // We rebuild the message manually here rather than trust the client to provide it.
    //
    // The nonce is read from state rather than taken from the message so the
    // proof can only ever be spent once: it is burned below, and a proof signed
    // against a burned nonce no longer rebuilds. The chain id binds the proof to
    // this chain. See types.AddressRegisterProofInput for why both are needed.
    proofNonce := cc.GetAddressProofNonce(msg.Address)
    hashInput := types.AddressRegisterProofInput(ctx.ChainID(), msg.PlayerId, msg.Address, proofNonce)
    k.logger.Info("Address Register", "hashInput", hashInput)

    // Decode the Signature from Hex Encoding
    decodedProofSignature, decodeErr := hex.DecodeString(msg.ProofSignature)
    if decodeErr != nil {
        k.logger.Error("Address Register Signature", "decodingError", decodeErr)
        return emptyResponse, decodeErr
    }

    if len(decodedProofSignature) < 64 {
        return emptyResponse, types.NewAddressValidationError(msg.Address, "signature_too_short")
    }

    // Proof needs to only be 64 bytes. Some systems provide a checksum byte on the end that ruins it all
    if (!pubKey.VerifySignature([]byte(hashInput), decodedProofSignature[:64])) {
         return emptyResponse, types.NewAddressValidationError(msg.Address, "signature_invalid")
    }

    // Burn the nonce this proof was signed against before anything is moved.
    // AddressRevoke deletes the association but leaves this row, so it is the
    // only thing that stops the same proof re-registering the address later and
    // sweeping it again.
    cc.IncrementAddressProofNonce(msg.Address)

	// Add the address and player index to the keeper
    cc.SetPlayerIndexForAddress(msg.Address, player.GetIndex())

	// Add the permission to the new address
    newAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    cc.PermissionAdd(newAddressPermissionId, types.Permission(msg.Permissions))


    // Move Funds
    primaryAcc, _   := sdk.AccAddressFromBech32(player.GetPrimaryAddress())
    newAcc, _   := sdk.AccAddressFromBech32(msg.Address)

    // Move Reactor Infusions over.
    //
    // Reassigns the player's delegations onto the newly registered address; the
    // source infusion is wound down under its previous owner and reclaimed, and
    // the destination infusion is rebuilt for this player from live staking.
    // Ahead of the coin sweep so that the staking rewards the transfer settles
    // are swept along with everything else.
    //
    // Strict: the incoming address belongs to the player either way, so a
    // refusal costs them nothing but a wait, and nobody else can create the
    // condition that triggers it.
    err = k.MoveDelegationsToAddress(ctx, cc, newAcc, player.GetPrimaryAddress(), DelegationTransferStrict)
    if err != nil {
        return emptyResponse, err
    }

    // Get Balance
    balances := k.bankKeeper.SpendableCoins(ctx, newAcc)

    // Transfer
    err = k.bankKeeper.SendCoins(ctx, newAcc, primaryAcc, balances)
    if err != nil {
        return emptyResponse, err
    }

	cc.CommitAll()
	return &types.MsgAddressRegisterResponse{}, nil
}
