package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) GuildApproveRegister(goCtx context.Context, msg *types.MsgGuildApproveRegister) (*types.MsgGuildApproveRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))

    _, _ = player, playerFound

    registeringPlayer, registeringPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    guild, guildFound := k.GetGuild(ctx, msg.GuildId)

    if (!guildFound) {
        // TODO error
        return &types.MsgGuildApproveRegisterResponse{}, nil
    }

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
