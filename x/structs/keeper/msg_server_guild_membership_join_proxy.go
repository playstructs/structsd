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
	emptyResponse := &types.MsgGuildMembershipResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// Look up requesting account
	proxyPlayer, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return emptyResponse, err
	}

	// look up destination guild
	guild := cc.GetGuild(proxyPlayer.GetGuildId())
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", proxyPlayer.GetGuildId())
	}

	// Decode the PubKey from hex Encoding
	k.logger.Info("Guild Join Proxy", "encodingString", msg.ProofPubKey)

	decodedProofPubKey, decodeErr := hex.DecodeString(msg.ProofPubKey)
	if decodeErr != nil {
		k.logger.Error("Guild Join Proxy Public Key", "decodingError", decodeErr)
		return emptyResponse, decodeErr
	}
	if len(decodedProofPubKey) != 33 {
		return emptyResponse, types.NewAddressValidationError(msg.Address, "proof_public_key_invalid")
	}

	// Convert provided pub key into a bech32 string (i.e., an address)
	address := types.PubKeyToBech32(decodedProofPubKey)

	if address != msg.Address {
		return emptyResponse, types.NewAddressValidationError(msg.Address, "proof_mismatch").WithPlayers(address, msg.Address)
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
		return emptyResponse, decodeErr
	}

	if len(decodedProofSignature) < 64 || len(decodedProofSignature) > 65 {
		return emptyResponse, types.NewAddressValidationError(msg.Address, "signature_invalid_length")
	}
	if !pubKey.VerifySignature([]byte(hashInput), decodedProofSignature[:64]) {
		return emptyResponse, types.NewAddressValidationError(msg.Address, "signature_invalid")
	}

	guildPermissionErr := guild.CanAddMembersByProxy(proxyPlayer)
	if guildPermissionErr != nil {
		return emptyResponse, guildPermissionErr
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
			return emptyResponse, types.NewObjectNotFoundError("substation", msg.SubstationId).WithContext("override substation")
		}

		// Since the Guild Entry Substation is being overridden, let's make
		// sure the ProxyPlayer actually have authority over this substation
		substationPermissionErr := substation.CanManageConnectionsBy(proxyPlayer)
		if substationPermissionErr != nil {
			return emptyResponse, substationPermissionErr
		}
		substationSet = true
	}

	if !substationSet {
		substation = cc.GetSubstation(guild.GetEntrySubstationId())
		if substation.CheckSubstation() != nil {
			return emptyResponse, types.NewObjectNotFoundError("substation", guild.GetEntrySubstationId()).WithContext("guild entry substation for " + guild.GetGuildId())
		}
	}

	/* Onboarding an address nobody has registered is permissionless on purpose:
	 * there is no player yet, so there is nobody whose authority could be
	 * bypassed and the proof of key possession is the whole of the consent.
	 *
	 * An address already bound to a player is a different operation wearing the
	 * same message. UpsertPlayer is an in-get rather than an insert, so it hands
	 * back that entire existing player and the mutations below reach the shared
	 * entity - guild, rank, substation connection, name, pfp. The direct join
	 * path demands PermGuildMembership of the acting address before any of that,
	 * and restricted secondary addresses exist precisely so a low-trust key can
	 * play without being able to move the player between guilds. Treating key
	 * possession as sufficient here let such a key do through the proxy exactly
	 * what it is barred from doing directly.
	 *
	 * This mirrors PermissionCheck's Layer 1 rather than calling it: the acting
	 * identity on the context is msg.Creator, the proxy, so a PermissionCheck
	 * here would test the wrong address's bits. The address that signed the proof
	 * is the one consenting, so it is the one that has to hold the bit.
	 */
	if cc.GetPlayerIndexFromAddress(msg.Address) > 0 {
		if !cc.PermissionHasAll(GetAddressPermissionIDBytes(msg.Address), types.PermGuildMembership) {
			return emptyResponse, types.NewPermissionError(
				"address", msg.Address, "", "",
				uint64(types.PermGuildMembership), types.PermissionName(types.PermGuildMembership),
			)
		}
	}

	// create new player
	player := cc.UpsertPlayer(msg.Address)

	if player.GetGuildId() != "" {
		return emptyResponse, types.NewGuildMembershipError(
			player.GetGuildId(), player.GetPlayerId(), "already_member",
		)
	}

	// Add player to the guild
	player.SetGuild(guild.GetGuildId())
	player.SetGuildRank(guild.GetEntryRank())

	// Connect player to the substation
	// Now let's get the player some power
	if player.GetSubstationId() == "" {
		// Connect Player to Substation
		player.MigrateSubstation(substation.GetSubstationId())
	}

	if msg.PlayerName != "" {
		if err := types.ValidatePlayerName(msg.PlayerName); err != nil {
			return emptyResponse, err
		}
		player.SetName(msg.PlayerName)
	}

	if msg.PlayerPfp != "" {
		if err := types.ValidatePfp(msg.PlayerPfp); err != nil {
			return emptyResponse, err
		}
		player.SetPfp(msg.PlayerPfp)
	}

	if msg.PlayerPfpClientRenderAttributes != "" {
		canonical, err := types.ValidatePfpClientRenderAttributes(msg.PlayerPfpClientRenderAttributes)
		if err != nil {
			return emptyResponse, err
		}
		player.SetPfpClientRenderAttributes(canonical)
	}

	// The proxy join has completely mostly successfully at this point
	// Increase the nonce of the player account to prevent replay of this signed message
	cc.SetGridAttributeIncrement(GetGridAttributeIDByObjectId(types.GridAttributeType_proxyNonce, player.GetPlayerId()), 1)

	cc.CommitAll()
	return &types.MsgGuildMembershipResponse{}, nil
}
