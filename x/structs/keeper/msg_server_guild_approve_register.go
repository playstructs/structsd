package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildApproveRegister(goCtx context.Context, msg *types.MsgGuildApproveRegister) (*types.MsgGuildApproveRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))


    if (!playerFound) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform guild action with non-player address (%s)", msg.Creator)
    }

    guild, guildFound := k.GetGuild(ctx, msg.GuildId)
    if (!guildFound) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Referenced Guild (%d) not found", guild.Id)
    }

    // Check to make sure the player has permissions on the guild
    if (!k.GuildPermissionHasOneOf(ctx, guild.Id, player.Id, types.GuildPermissionRegisterPlayer)) {
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%d) has no Player Registration permissions ", player.Id)
    }

    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    // Make sure the address calling this has Associate permissions
    if (playerPermissions&types.AddressPermissionManageGuild == 0) {
        // TODO permission error
        return &types.MsgGuildApproveRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }


    registeringPlayer, registeringPlayerFound := k.GetPlayer(ctx, msg.PlayerId)



    if (registeringPlayerFound) {
        if (msg.Approve) {
            // TODO permission checking to see if this specific account has the ability to grant these permissions

            k.GuildApproveRegisterRequest(ctx, guild, registeringPlayer)
        } else {
            k.GuildDenyRegisterRequest(ctx, guild, registeringPlayer)
        }
    }

	return &types.MsgGuildApproveRegisterResponse{}, nil
}
