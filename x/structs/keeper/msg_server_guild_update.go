package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdate(goCtx context.Context, msg *types.MsgGuildUpdate) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    var playerAddress sdk.AccAddress
    playerAddress, _ = sdk.AccAddressFromBech32(msg.Creator)

    var validatorAddress sdk.ValAddress
    validatorAddress = playerAddress.Bytes()

    reactorBytes, _ := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
    _, reactorFound := k.GetReactorByBytes(ctx, reactorBytes, true)

    if (!reactorFound) {
        return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrReactorRequired, "Guild creation requires Reactor but none associated with %s", msg.Creator)
    }



    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        // should really never get here as player creation is triggered
        // during reactor initialization
        return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Guild creation requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)


    guild, guildFound := k.GetGuild(ctx, msg.Id)
    if (!guildFound) {
            return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Guild wasn't found. Can't update that which does not exist", msg.Id)
    }

    if (!k.GuildPermissionHasOneOf(ctx, msg.Id, player.Id, types.GuildPermissionUpdate)) {
        return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildUpdate, "Calling player (%d) has no permissions to update guild", player.Id)
    }

    if ((msg.Owner > 0) && (guild.Owner != msg.Owner)) {
        _, guildOwnerFound := k.GetPlayer(ctx, msg.Owner)
        if (!guildOwnerFound) {
            return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildUpdate, "Guild could not change to new owner (%d) because they weren't found", msg.Owner)
        }
        guild.SetOwner(msg.Owner)
    }

    if (msg.EntrySubstationId > 0 ) {
        // check that the calling player has substation permissions
        if (!k.SubstationPermissionHasOneOf(ctx, msg.EntrySubstationId, player.Id, types.SubstationPermissionConnectPlayer)) {
            return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%d) has no Substation Connect Player permissions ", player.Id)
        }
        guild.SetEntrySubstationId(msg.EntrySubstationId)
    }

    if (msg.Endpoint != "") {
        guild.SetEndpoint(msg.Endpoint)
    }

    if (msg.GuildJoinType != guild.GuildJoinType) {
        guild.SetGuildJoinType(msg.GuildJoinType)
    }

    if (msg.InfusionJoinMinimum != guild.InfusionJoinMinimum) {
        guild.SetInfusionJoinMinimum(msg.InfusionJoinMinimum)
    }

    k.SetGuild(ctx, guild)


	return &types.MsgGuildUpdateResponse{}, nil
}
