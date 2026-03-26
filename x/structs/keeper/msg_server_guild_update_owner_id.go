package keeper

import (
	"context"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) GuildUpdateOwnerId(goCtx context.Context, msg *types.MsgGuildUpdateOwnerId) (*types.MsgGuildUpdateResponse, error) {
    emptyResponse := &types.MsgGuildUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_update_owner")
	}

	guild := cc.GetGuild(msg.GuildId)
	if guild.CheckGuild() != nil {
		return emptyResponse, types.NewObjectNotFoundError("guild", msg.GuildId)
	}

    permissionErr := guild.CanTransferOwnershipBy(player)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

	if guild.GetGuild().Owner != msg.Owner {
		newOwner, _ := cc.GetPlayer(msg.Owner)
		if newOwner.CheckPlayer() != nil {
			return emptyResponse, types.NewObjectNotFoundError("player", msg.Owner)
		}

		guild.SetOwner(msg.Owner)
	}

	cc.CommitAll()
	return &types.MsgGuildUpdateResponse{}, nil
}
