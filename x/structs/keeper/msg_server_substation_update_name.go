package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationUpdateName(goCtx context.Context, msg *types.MsgSubstationUpdateName) (*types.MsgSubstationUpdateResponse, error) {
	emptyResponse := &types.MsgSubstationUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "substation_update_name")
	}

	substation := cc.GetSubstation(msg.SubstationId)
	if substation.CheckSubstation() != nil {
		return emptyResponse, types.NewObjectNotFoundError("substation", msg.SubstationId)
	}

	permissionErr := substation.CanUpdateUGCBy(player)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	if err := types.ValidateEntityName(msg.Name); err != nil {
		return emptyResponse, err
	}

	oldName := substation.GetName()
	substation.SetName(msg.Name)
	emitUGCModerationEventIfActorIsNotOwner(ctx, substation, player, types.UGCFieldName, oldName, msg.Name)

	cc.CommitAll()
	return &types.MsgSubstationUpdateResponse{}, nil
}
