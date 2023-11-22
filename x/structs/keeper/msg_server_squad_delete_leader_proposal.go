package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadDeleteLeaderProposal(goCtx context.Context, msg *types.MsgSquadDeleteLeaderProposal) (*types.MsgSquadDeleteLeaderProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    // look up transaction creator player object
    txPlayerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (txPlayerId == 0) {
        return &types.MsgSquadDeleteLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Squad creation requires Player account but none associated with %s", msg.Creator)
    }
    txPlayer, _ := k.GetPlayer(ctx, txPlayerId)

	// look up destination squad
	squad, squadFound := k.GetSquad(ctx, msg.SquadId)

    if (!squadFound) {
        return &types.MsgSquadDeleteLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadNotFound, "Referenced Squad (%d) not found", msg.SquadId)
    }

    // Calling address (msg.Creator) must have permissions to perform the Squad Management task
    // AddressPermissionManageSquad
    if (!k.AddressPermissionHasOneOf(ctx, msg.Creator, types.AddressPermissionManageSquad)) {
        return &types.MsgSquadDeleteLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling Address (%s) must have Squad Management permissions", msg.Creator)
    }

    // Calling player (txPlayer) needs to have certain permissions to complete the task
    // either GuildPermissionSquadUpdate or SquadPermissionUpdateLeader
    if ((!k.GuildPermissionHasOneOf(ctx, squad.GuildId, txPlayer.Id, types.GuildPermissionSquadUpdate)) && (!k.SquadPermissionHasOneOf(ctx, squad.Id, txPlayer.Id, types.SquadPermissionUpdateLeader))) {
        return &types.MsgSquadDeleteLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadLeaderProposal, "Calling player (%d) does not have Squad Update permissions from Guild (%d) or Squad (%d)", txPlayer.Id, squad.GuildId, squad.Id)
    }

    k.SquadDeleteLeaderProposalRequest(ctx, squad)

	return &types.MsgSquadDeleteLeaderProposalResponse{}, nil
}
