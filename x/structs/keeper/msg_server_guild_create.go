package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildCreate(goCtx context.Context, msg *types.MsgGuildCreate) (*types.MsgGuildCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}


    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return nil, sdkerrors.Wrapf(types.ErrPlayerRequired, "Guild creation requires Player account but none associated with %s", msg.Creator)
    }
    //player, _ = k.GetPlayer(ctx, playerId)

	guild := types.CreateEmptyGuild()
	guild.SetEndpoint(msg.Endpoint)
	guild.SetCreator(msg.Creator)
	guild.SetOwner(playerId)

	newGuildId := k.AppendGuild(ctx, guild)
    k.GuildPermissionAdd(ctx, newGuildId, playerId, types.GuildPermissionAll)


	return &types.MsgGuildCreateResponse{GuildId: newGuildId}, nil
}
