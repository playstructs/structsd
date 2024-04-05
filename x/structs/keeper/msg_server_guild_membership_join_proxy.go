package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"


	//cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	crypto "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	 "crypto/sha256"
      "golang.org/x/crypto/ripemd160"
      //"encoding/base64"

      "github.com/cosmos/btcutil/bech32"
)

func (k msgServer) GuildMembershipJoinProxy(goCtx context.Context, msg *types.MsgGuildMembershipJoinProxy) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Look up requesting account
	proxyPlayer := k.UpsertPlayer(ctx, msg.Creator, true)

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, proxyPlayer.GuildId)
    if (!guildFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Referenced Guild (%s) not found", guild.Id)
    }


    // Thank you Sunny
    // https://gist.github.com/arnabghose997/f2d33955d5359de255055d6ce8c619f7
    // Hash pubKeyBytes as: RIPEMD160(SHA256(public_key_bytes))
	pubKeySha256Hash := sha256.Sum256(msg.ProofPubKey)
	ripemd160hash := ripemd160.New()
	ripemd160hash.Write(pubKeySha256Hash[:])
	addressBytes := ripemd160hash.Sum(nil)

    // Convert addressBytes into a bech32 string
	address := toBech32("structs", addressBytes)

    if (address != msg.Address) {
         return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Proof mismatch for %s vs %s vs %s", address, msg.Address)
    }



    pubKey := crypto.PubKey{}
    pubKey.Key = msg.ProofPubKey


    // We rebuild the message manually here rather than trust the client to provide it
    // TODO, we need to eventually add replay protection here as right now this
    // would allow guilds to use the same proof repeatedly.
    hashInput := "GUILD" + guild.Id + "ADDRESS" + msg.Address
    if (!pubKey.VerifySignature([]byte(hashInput), msg.ProofSignature)) {
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
	    substation, substationFound = k.GetSubstation(ctx, msg.SubstationId, true)
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
        substation, substationFound = k.GetSubstation(ctx, guild.EntrySubstationId, true)
        if (!substationFound) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Entry Substation (%s) for Guild (%s) not found", guild.EntrySubstationId, guild.Id)
        }
    }

	// create new player
    player := k.UpsertPlayer(ctx, msg.Address, true)


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

	return &types.MsgGuildMembershipResponse{}, nil
}

// Thank you Sunny
// https://gist.github.com/arnabghose997/f2d33955d5359de255055d6ce8c619f7
// Code courtesy: https://github.com/cosmos/cosmos-sdk/blob/90c9c9a9eb4676d05d3f4b89d9a907bd3db8194f/types/bech32/bech32.go#L10
// TODO move this to utils
func toBech32(addrPrefix string, addrBytes []byte) string {
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