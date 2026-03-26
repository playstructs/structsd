package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipKick(goCtx context.Context, msg *types.MsgGuildMembershipKick) (*types.MsgGuildMembershipResponse, error) {
    emptyResponse := &types.MsgGuildMembershipResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    guildMembershipApplication, guildMembershipApplicationError := cc.GetGuildMembershipKickCache(callingPlayer, msg.GuildId, msg.PlayerId)
    if guildMembershipApplicationError != nil {
        return emptyResponse, guildMembershipApplicationError
    }

    guildMembershipApplicationError = guildMembershipApplication.Kick()
    if guildMembershipApplicationError != nil {
        return emptyResponse, guildMembershipApplicationError
    }

	cc.CommitAll()
	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}
