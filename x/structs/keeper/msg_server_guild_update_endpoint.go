package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildUpdateEndpoint(goCtx context.Context, msg *types.MsgGuildUpdateEndpoint) (*types.MsgGuildUpdateResponse, error) {
    emptyResponse := &types.MsgGuildUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_endpoint")
    }

    guild := cc.GetGuild(msg.GuildId)
    if guild.CheckGuild() != nil {
        return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
    }

    permissionErr := guild.CanUpdateEndpointBy(player)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    guild.SetEndpoint(msg.Endpoint)

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
