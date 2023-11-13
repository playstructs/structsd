package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadApproveLeaderProposal(goCtx context.Context, msg *types.MsgSquadApproveLeaderProposal) (*types.MsgSquadApproveLeaderProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    txPlayer, txPlayerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))


    if (!txPlayerFound) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform guild action with non-player address (%s)", msg.Creator)
    }

    squad, squadFound := k.GetSquad(ctx, msg.SquadId)
    if (!squadFound) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Referenced Squad (%d) not found", squad.Id)
    }

    leaderPlayer, leaderPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!leaderPlayerFound) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Proposed Leader (%d) does not exist", msg.PlayerId)
    }

   // Check Permissions
   if (txPlayer.Id != leaderPlayer.Id) {
        // Does the calling player have sudo on the leader?
        if (!k.PlayerPermissionHasOneOf(ctx, leaderPlayer.Id, txPlayer.Id, types.PlayerPermissionSquad)) {
            return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquad, "Calling player (%d) has no Sudo Squad permissions on Proposed Leader (%d)", txPlayer.Id, msg.PlayerId)
        }
   }

    // check on address permissions
    // AddressPermissionManageSquad
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionManageSquad) == 0) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquad, "Calling address (%s) has no Squad permissions ", msg.Creator)
    }


    // Confirm the Results
    if (msg.Approve) {
        if (!k.SquadApproveLeaderProposalRequest(ctx, squad, leaderPlayer)) {
            return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Error during proposal approval")
        }
    } else {
        k.SquadDenyLeaderProposalRequest(ctx, squad, leaderPlayer)
    }


	return &types.MsgSquadApproveLeaderProposalResponse{}, nil
}
