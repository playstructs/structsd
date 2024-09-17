package keeper

import (
	"context"
    "fmt"
    "encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

	crypto "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

    //cometbftcrypto "github.com/cometbft/cometbft/crypto"
)

func (k msgServer) GuildMembershipJoinProxy(goCtx context.Context, msg *types.MsgGuildMembershipJoinProxy) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// Look up requesting account
	proxyPlayer := k.UpsertPlayer(ctx, msg.Creator)

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, proxyPlayer.GuildId)
    if (!guildFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Referenced Guild (%s) not found", guild.Id)
    }

	// Decode the PubKey from hex Encoding
    fmt.Println("Encoding string:", msg.ProofPubKey)

    decodedProofPubKey, decodeErr := hex.DecodeString(msg.ProofPubKey)
    if decodeErr != nil {
        fmt.Println("Error decoding string:", decodeErr)
    }

    // Convert provided pub key into a bech32 string (i.e., an address)
	address := types.PubKeyToBech32(decodedProofPubKey)

    if (address != msg.Address) {
         return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Proof mismatch for %s vs %s vs %s", address, msg.Address)
    }

    pubKey := crypto.PubKey{}
    pubKey.Key = decodedProofPubKey

    // Check to see if the account has ever been used before
    // If it has, then grab the nonce to make sure there is not a replay attack being taken against the player
    //
    // A playerIndex of 0 should never return anything other than a nonce of 0
    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Address)
    nonce := k.GetGridAttribute(ctx, GetGridAttributeID(types.GridAttributeType_proxyNonce, types.ObjectType_player, playerIndex))

    // We rebuild the message manually here rather than trust the client to provide it
    hashInput := fmt.Sprintf("GUILD%sADDRESS%sNONCE%d", guild.Id, msg.Address, nonce)

    /*
    fmt.Printf("Hash: %s \n", hashInput)
    hashDigest := cometbftcrypto.Sha256([]byte(hashInput))
    fmt.Printf("\n Digest ", hashDigest)
    fmt.Printf("\n Digest %s \n", hex.EncodeToString(hashDigest))
    fmt.Printf("Proof", msg.ProofSignature)
    fmt.Printf("Proof\n")

    fmt.Printf("Digest Length: %d \n", len(hashDigest))
    fmt.Printf("Proof Length: %d \n", len(msg.ProofSignature))
    */

    // Decode the Signature from Hex Encoding
    decodedProofSignature, decodeErr := hex.DecodeString(msg.ProofSignature)
    if decodeErr != nil {
        fmt.Println("Error decoding string:", decodeErr)
    }

    // Proof needs to only be 64 characters. Some systems provide a checksum bit on the end that ruins it all
    if (!pubKey.VerifySignature([]byte(hashInput), decodedProofSignature[:64])) {
         return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Proof signature verification failure")
    }

    guildObjectPermissionId := GetObjectPermissionIDBytes(guild.Id, proxyPlayer.Id)
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    // Check to make sure the player has permissions on the guild
    if (!k.PermissionHasOneOf(ctx, guildObjectPermissionId, types.PermissionAssociations)) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Registration permissions ", proxyPlayer.Id)
    }

    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssociations)) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }


    var substation types.Substation
    substationFound := false

	/* Look up destination substation
	 *
	 * We're going to try and load up the substation override first
	 * and if that doesn't exist, we'll go load up the regular
	 * guild entry substation.
	 *
	 * Proxy player needs permissions on the override but the default
	 * entry substation will always work.
	 */

	if (msg.SubstationId != "") {
	    substation, substationFound = k.GetSubstation(ctx, msg.SubstationId)
        if (!substationFound) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Provided Substation Override (%s) not found", msg.SubstationId)
        }

        // Since the Guild Entry Substation is being overridden, let's make
        // sure the ProxyPlayer actually have authority over this substation
        substationObjectPermissionId := GetObjectPermissionIDBytes(substation.Id, proxyPlayer.Id)
        if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%d) has no Player Connect permissions on Substation (%s) used as override", proxyPlayer.Id, substation.Id)
        }
	}

    if (!substationFound) {
        substation, substationFound = k.GetSubstation(ctx, guild.EntrySubstationId)
        if (!substationFound) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Entry Substation (%s) for Guild (%s) not found", guild.EntrySubstationId, guild.Id)
        }
    }

	// create new player
    player := k.UpsertPlayer(ctx, msg.Address)


    if (player.GuildId != "") {
        // TODO new guild setting that dictates what to do when a player leaves
            // If already in a guild, leave permissions as-is?
            // Force disconnection of Substation?

        // look up old guild
        oldGuild, _ := k.GetGuild(ctx, player.GuildId)

        // Let's only disconnect the player if it's the main substation for the guild
        // Otherwise it might be there own substation and maybe they don't really want
        // that fucked with. Could also throw a flag in the calling message to force this.
        if (player.SubstationId != "" && player.SubstationId == oldGuild.EntrySubstationId) {
            player, _ = k.SubstationDisconnectPlayer(ctx, player)
        }
    }

    // Add player to the guild
    player.GuildId = guild.Id

    // Connect player to the substation
    // Now let's get the player some power
    if (player.SubstationId == "") {
        // Connect Player to Substation
        k.SubstationConnectPlayer(ctx, substation, player)
    }

    // The proxy join has completely mostly successfully at this point
    // Increase the nonce of the player account to prevent replay of this signed message
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_proxyNonce, player.Id), 1)

	return &types.MsgGuildMembershipResponse{}, nil
}

