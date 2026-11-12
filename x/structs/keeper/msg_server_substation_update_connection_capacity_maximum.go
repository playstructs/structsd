package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationUpdateConnectionCapacityMaximum(goCtx context.Context, msg *types.MsgSubstationUpdateConnectionCapacityMaximum) (*types.MsgSubstationUpdateResponse, error) {
	emptyResponse := &types.MsgSubstationUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetSigningPlayer(msg.Creator)
	if err != nil {
		return emptyResponse, err
	}
	substation, err := cc.GetExistingSubstation(msg.SubstationId)
	if err != nil {
		return emptyResponse, err
	}
	if err := substation.CanUpdateConnectionCapacityMaximumBy(player); err != nil {
		return emptyResponse, err
	}

	substation.SetConnectionCapacityMaximum(msg.ConnectionCapacityMaximum)
	cc.CommitAll()
	return emptyResponse, nil
}
