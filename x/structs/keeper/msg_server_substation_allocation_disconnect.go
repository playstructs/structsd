package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
    "strconv"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)


// Can't decide if this should be SubstationAllocationDisconnect, or AllocationDisconnect - since there are no other types of disconnections
func (k msgServer) SubstationAllocationDisconnect(goCtx context.Context, msg *types.MsgSubstationAllocationDisconnect) (*types.MsgSubstationAllocationDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}


	allocation, allocationFound  := k.GetAllocation(ctx, msg.AllocationId)
    if (!allocationFound){
        allocationId := strconv.FormatUint(msg.AllocationId, 10)
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationNotFound, "allocation (%s) not found", allocationId)
    }


	substation, substationFound  := k.GetSubstation(ctx, allocation.DestinationId)
    if (!substationFound){
        // Really shouldn't happen
        substationId := strconv.FormatUint(allocation.DestinationId, 10)
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "destination substation (%s) not found", substationId)
    }


    newEnergy := k.SubstationDecrementEnergy(ctx, substation.Id, allocation.Power)
    k.CascadeSubstationAllocationFailure(ctx, substation)

    allocation.Disconnect()
    k.SetAllocation(ctx, allocation)

	return &types.MsgSubstationAllocationDisconnectResponse{}, nil
}
