package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildCreate(goCtx context.Context, msg *types.MsgGuildCreate) (*types.MsgGuildCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var playerAddress sdk.AccAddress
    playerAddress, _ = sdk.AccAddressFromBech32(msg.Creator)

    var validatorAddress sdk.ValAddress
    validatorAddress = playerAddress.Bytes()

    reactorBytes, _ := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
    reactor := cc.GetReactor(string(reactorBytes))

    if reactor.CheckReactor() != nil {
        return &types.MsgGuildCreateResponse{}, types.NewReactorError("guild_create", "required").WithAddress(msg.Creator, "validator")
    }

    // Currently, no real reason to do permission checks that the player can
    // add the reactor, since the player account IS the reactor
        // TODO
        // Although, this should be changed so that if an account
        // that includes a validator as an associated address, it's able to perform this step


    player, playerErr := cc.GetPlayerByAddress(msg.Creator)
    if playerErr != nil {
        // should really never get here as player creation is triggered
        // during reactor initialization
        return &types.MsgGuildCreateResponse{}, types.NewPlayerRequiredError(msg.Creator, "guild_create")
    }

    if (msg.EntrySubstationId != "") {

        // Check that the Substation exists
        substation := cc.GetSubstation(msg.EntrySubstationId)
        if substation.CheckSubstation() != nil {
            return &types.MsgGuildCreateResponse{}, types.NewObjectNotFoundError("substation", msg.EntrySubstationId)
        }

        // check that the calling player has substation permissions
        substationObjectPermissionId := GetObjectPermissionIDBytes(msg.EntrySubstationId, player.GetPlayerId())
        if (!cc.PermissionHasOneOf(substationObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgGuildCreateResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "substation", msg.EntrySubstationId, uint64(types.PermissionGrid), "substation_connect")
        }
    }

    // TODO Fix Guild Creation
    guild := k.AppendGuild(ctx, msg.Endpoint, msg.EntrySubstationId, reactor.GetReactor(), player.GetPlayer())

    player.SetGuild(guild.Id)
    reactor.SetGuild(guild.Id)

	return &types.MsgGuildCreateResponse{GuildId: guild.Id}, nil
}
