package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerUpdateName(goCtx context.Context, msg *types.MsgPlayerUpdateName) (*types.MsgPlayerUpdateResponse, error) {
	emptyResponse := &types.MsgPlayerUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "player_update_name")
	}

	player, err := cc.GetPlayer(msg.PlayerId)
	if err != nil {
		return emptyResponse, types.NewObjectNotFoundError("player", msg.PlayerId)
	}

	permissionErr := player.CanUpdateUGCBy(activePlayer)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	if err := types.ValidatePlayerName(msg.Name); err != nil {
		return emptyResponse, err
	}

	player.SetName(msg.Name)

	cc.CommitAll()
	return &types.MsgPlayerUpdateResponse{}, nil
}
