package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadDeleteInvite(goCtx context.Context, msg *types.MsgSquadDeleteInvite) (*types.MsgSquadDeleteInviteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    // look up transaction creator player object
    txPlayerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (txPlayerId == 0) {
        return &types.MsgSquadDeleteInviteResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Squad management requires Player account but none associated with %s", msg.Creator)
    }
    txPlayer, _ := k.GetPlayer(ctx, txPlayerId)

    // look up target player object
    targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        return &types.MsgSquadDeleteInviteResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Referenced target player (%d) not found", msg.PlayerId)
    }


	// look up destination squad
	squad, squadFound := k.GetSquad(ctx, msg.SquadId)

    if (!squadFound) {
        return &types.MsgSquadDeleteInviteResponse{}, sdkerrors.Wrapf(types.ErrSquadNotFound, "Referenced Squad (%d) not found", msg.SquadId)
    }

    if (squad.Id == targetPlayer.SquadId) {
        return &types.MsgSquadDeleteInviteResponse{}, sdkerrors.Wrapf(types.ErrSquadPlayerCannotSquadHarder, "Proposed Player (%d) already in the Squad (%d)", msg.PlayerId, msg.SquadId)
    }

    // Calling address (msg.Creator) must have permissions to perform the Squad Management task
    // AddressPermissionManageSquad
    if (!k.AddressPermissionHasOneOf(ctx, msg.Creator, types.AddressPermissionManageSquad)) {
        return &types.MsgSquadDeleteInviteResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling Address (%s) must have Squad Management permissions", msg.Creator)
    }

    // Calling player (txPlayer) needs to have certain permissions to complete the task
    // either GuildPermissionSquadUpdate or SquadInvite
    if ((!k.GuildPermissionHasOneOf(ctx, squad.GuildId, txPlayer.Id, types.GuildPermissionSquadUpdate)) && (!k.SquadPermissionHasOneOf(ctx, squad.Id, txPlayer.Id, types.SquadInvite))) {
        return &types.MsgSquadDeleteInviteResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadLeaderProposal, "Calling player (%d) does not have Squad Update permissions from Guild (%d) or Squad (%d)", txPlayer.Id, squad.GuildId, squad.Id)
    }

    k.SquadDeleteInvite(ctx, squad, targetPlayer)

	return &types.MsgSquadDeleteInviteResponse{}, nil
}
