package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateSquadJoinTypeMinimum(goCtx context.Context, msg *types.MsgGuildUpdateSquadJoinTypeMinimum) (*types.MsgGuildUpdateResponse, error) {
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

    if (msg.SquadJoinTypeMinimum != guild.SquadJoinTypeMinimum) {
        guild.SetSquadJoinTypeMinimum(msg.SquadJoinTypeMinimum)
        k.SetGuild(ctx, guild)
    }




	return &types.MsgGuildUpdateResponse{}, nil
}
