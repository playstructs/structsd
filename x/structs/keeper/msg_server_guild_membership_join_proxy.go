package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	crypto "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	//cometbftcrypto "github.com/cometbft/cometbft/crypto"
)

func (k msgServer) GuildMembershipJoinProxy(goCtx context.Context, msg *types.MsgGuildMembershipJoinProxy) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// Look up requesting account
	proxyPlayer := cc.UpsertPlayer(msg.Creator)

	// look up destination guild
	guild := cc.GetGuild(proxyPlayer.GetGuildId())
	if guild.CheckGuild() != nil {
		return &types.MsgGuildMembershipResponse{}, types.NewObjectNotFoundError("guild", proxyPlayer.GetGuildId())
	}

	// Decode the PubKey from hex Encoding
	k.logger.Info("Guild Join Proxy", "encodingString", msg.ProofPubKey)

	decodedProofPubKey, decodeErr := hex.DecodeString(msg.ProofPubKey)
	if decodeErr != nil {
	    k.logger.Error("Guild Join Proxy Public Key", "decodingError", decodeErr)
	}

	// Convert provided pub key into a bech32 string (i.e., an address)
	address := types.PubKeyToBech32(decodedProofPubKey)

	if address != msg.Address {
		return &types.MsgGuildMembershipResponse{}, types.NewAddressValidationError(msg.Address, "proof_mismatch").WithPlayers(address, msg.Address)
	}

	pubKey := crypto.PubKey{}
	pubKey.Key = decodedProofPubKey

	// Check to see if the account has ever been used before
	// If it has, then grab the nonce to make sure there is not a replay attack being taken against the player
	//
	// A playerIndex of 0 should never return anything other than a nonce of 0
	playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Address)
	nonce := cc.GetGridAttribute(GetGridAttributeID(types.GridAttributeType_proxyNonce, types.ObjectType_player, playerIndex))

	// We rebuild the message manually here rather than trust the client to provide it
	hashInput := fmt.Sprintf("GUILD%sADDRESS%sNONCE%d", guild.GetGuildId(), msg.Address, nonce)

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
	    k.logger.Error("Guild Join Proxy Signature", "decodingError", decodeErr)
	}

	// Proof needs to only be 64 characters. Some systems provide a checksum bit on the end that ruins it all
	if !pubKey.VerifySignature([]byte(hashInput), decodedProofSignature[:64]) {
		return &types.MsgGuildMembershipResponse{}, types.NewAddressValidationError(msg.Address, "signature_invalid")
	}

	guildObjectPermissionId := GetObjectPermissionIDBytes(guild.GetGuildId(), proxyPlayer.GetPlayerId())
	addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

	// Check to make sure the player has permissions on the guild
	if !cc.PermissionHasOneOf(guildObjectPermissionId, types.PermissionAssociations) {
		return &types.MsgGuildMembershipResponse{}, types.NewPermissionError("player", proxyPlayer.GetPlayerId(), "guild", guild.GetGuildId(), uint64(types.PermissionAssociations), "register_player")
	}

	// Make sure the address calling this has Associate permissions
	if !cc.PermissionHasOneOf(addressPermissionId, types.PermissionAssociations) {
		return &types.MsgGuildMembershipResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssociations), "guild_management")
	}

	var substation *SubstationCache
    substationSet := false
	/* Look up destination substation
	 *
	 * We're going to try and load up the substation override first
	 * and if that doesn't exist, we'll go load up the regular
	 * guild entry substation.
	 *
	 * Proxy player needs permissions on the override but the default
	 * entry substation will always work.
	 */

	if msg.SubstationId != "" {
		substation = cc.GetSubstation(msg.SubstationId)
		if substation.CheckSubstation() != nil {
			return &types.MsgGuildMembershipResponse{}, types.NewObjectNotFoundError("substation", msg.SubstationId).WithContext("override substation")
		}

		// Since the Guild Entry Substation is being overridden, let's make
		// sure the ProxyPlayer actually have authority over this substation
		substationObjectPermissionId := GetObjectPermissionIDBytes(substation.GetSubstationId(), proxyPlayer.GetPlayerId())
		if !cc.PermissionHasOneOf(substationObjectPermissionId, types.PermissionGrid) {
			return &types.MsgGuildMembershipResponse{}, types.NewPermissionError("player", proxyPlayer.GetPlayerId(), "substation", substation.GetSubstationId(), uint64(types.PermissionGrid), "player_connect")
		}
		substationSet = true
	}

	if !substationSet {
		substation = cc.GetSubstation(guild.GetEntrySubstationId())
		if substation.CheckSubstation() != nil {
			return &types.MsgGuildMembershipResponse{}, types.NewObjectNotFoundError("substation", guild.GetEntrySubstationId()).WithContext("guild entry substation for " + guild.GetGuildId())
		}
	}

	// create new player
	player := cc.UpsertPlayer(msg.Address)

	if player.GetGuildId() != "" {
		// TODO new guild setting that dictates what to do when a player leaves
		// If already in a guild, leave permissions as-is?
		// Force disconnection of Substation?

		// look up old guild
		oldGuild := cc.GetGuild(player.GetGuildId())

		// Let's only disconnect the player if it's the main substation for the guild
		// Otherwise it might be there own substation and maybe they don't really want
		// that fucked with. Could also throw a flag in the calling message to force this.
		if player.GetSubstationId() != "" && player.GetSubstationId() == oldGuild.GetEntrySubstationId() {
			player.DisconnectSubstation()
		}
	}

	// Add player to the guild
	player.SetGuild(guild.GetGuildId())

	// Connect player to the substation
	// Now let's get the player some power
	if player.GetSubstationId() == "" {
		// Connect Player to Substation
		player.MigrateSubstation(substation.GetSubstationId())
	}

	// The proxy join has completely mostly successfully at this point
	// Increase the nonce of the player account to prevent replay of this signed message
	cc.SetGridAttributeIncrement(GetGridAttributeIDByObjectId(types.GridAttributeType_proxyNonce, player.GetPlayerId()), 1)

	cc.CommitAll()
	return &types.MsgGuildMembershipResponse{}, nil
}
