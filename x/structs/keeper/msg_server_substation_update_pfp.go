package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationUpdatePfp(goCtx context.Context, msg *types.MsgSubstationUpdatePfp) (*types.MsgSubstationUpdateResponse, error) {
	emptyResponse := &types.MsgSubstationUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "substation_update_pfp")
	}

	substation := cc.GetSubstation(msg.SubstationId)
	if substation.CheckSubstation() != nil {
		return emptyResponse, types.NewObjectNotFoundError("substation", msg.SubstationId)
	}

	permissionErr := substation.CanUpdateUGCBy(player)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	if err := types.ValidatePfp(msg.Pfp); err != nil {
		return emptyResponse, err
	}

	substation.SetPfp(msg.Pfp)

	cc.CommitAll()
	return &types.MsgSubstationUpdateResponse{}, nil
}
