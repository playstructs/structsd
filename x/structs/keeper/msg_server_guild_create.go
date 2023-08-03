package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildCreate(goCtx context.Context, msg *types.MsgGuildCreate) (*types.MsgGuildCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	guild := types.CreateEmptyGuild()
	guild.SetEndpoint(msg.Endpoint)
	guild.SetCreator(msg.Creator)

    // Check if Creator is associated with a validator

    // Check to see if Creator is associated with a player
    // if not, create that player account




	newGuildId := k.AppendGuild(ctx, guild)


	return &types.MsgGuildCreateResponse{GuildId: newGuildId}, nil
}
