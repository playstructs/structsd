package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateEntrySubstationId(goCtx context.Context, msg *types.MsgGuildUpdateEntrySubstationId) (*types.MsgGuildUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
     return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Guild update requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)

    guild, guildFound := k.GetGuild(ctx, msg.Id)
    if (!guildFound) {
         return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Guild wasn't found. Can't update that which does not exist", msg.Id)
    }

    if (!k.GuildPermissionHasOneOf(ctx, msg.Id, player.Id, types.GuildPermissionUpdate)) {
     return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrGuildUpdate, "Calling player (%d) has no permissions to update guild", player.Id)
    }

    if (msg.EntrySubstationId > 0 ) {
        // check that the calling player has substation permissions
        if (!k.SubstationPermissionHasOneOf(ctx, msg.EntrySubstationId, player.Id, types.SubstationPermissionConnectPlayer)) {
            return &types.MsgGuildUpdateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSubstationPlayerConnect, "Calling player (%d) has no Substation Connect Player permissions ", player.Id)
        }
        guild.SetEntrySubstationId(msg.EntrySubstationId)
    }

    k.SetGuild(ctx, guild)


	return &types.MsgGuildUpdateResponse{}, nil
}
