package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerUpdatePfpClientRenderAttributes(goCtx context.Context, msg *types.MsgPlayerUpdatePfpClientRenderAttributes) (*types.MsgPlayerUpdateResponse, error) {
	emptyResponse := &types.MsgPlayerUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "player_update_pfp_client_render_attributes")
	}

	player, err := cc.GetPlayer(msg.PlayerId)
	if err != nil {
		return emptyResponse, types.NewObjectNotFoundError("player", msg.PlayerId)
	}

	permissionErr := player.CanUpdateUGCBy(activePlayer)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	canonical, err := types.ValidatePfpClientRenderAttributes(msg.PfpClientRenderAttributes)
	if err != nil {
		return emptyResponse, err
	}

	oldAttributes := player.GetPfpClientRenderAttributes()
	player.SetPfpClientRenderAttributes(canonical)
	emitUGCModerationEventIfActorIsNotOwner(ctx, player, activePlayer, types.UGCFieldPfpClientRenderAttributes, oldAttributes, canonical)

	cc.CommitAll()
	return &types.MsgPlayerUpdateResponse{}, nil
}
