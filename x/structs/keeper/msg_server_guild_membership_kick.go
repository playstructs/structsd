package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipKick(goCtx context.Context, msg *types.MsgGuildMembershipKick) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


    callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, msg.Creator)
    if err != nil {
        return &types.MsgGuildMembershipResponse{}, err
    }

    // Use cache permission methods
    callingPlayerPermissionError := callingPlayer.CanBeAdministratedBy(msg.Creator, types.PermissionAssociations)
    if callingPlayerPermissionError != nil {
        return &types.MsgGuildMembershipResponse{}, callingPlayerPermissionError
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    guildMembershipApplication, guildMembershipApplicationError := k.GetGuildMembershipKickCache(ctx, &callingPlayer, msg.GuildId, msg.PlayerId)
    if guildMembershipApplicationError != nil {
        return &types.MsgGuildMembershipResponse{}, guildMembershipApplicationError
    }

    guildMembershipApplicationError = guildMembershipApplication.Kick()
    if guildMembershipApplicationError != nil {
        return &types.MsgGuildMembershipResponse{}, guildMembershipApplicationError
    }

	guildMembershipApplication.Commit()

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}
