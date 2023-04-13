package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
    "strconv"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)


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


    /*
     * This section is a little repetitive due to the fact that I can't
     * just have a generic source variable that can switch between types
     *
     */
    switch allocation.SourceType {
        case types.ObjectType_substation:
            source, sourceFound := k.GetSubstation(ctx, allocation.SourceId)

            if (!sourceFound){
                sourceId := strconv.FormatUint(allocation.SourceId, 10)
                return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s) used for allocation not found", allocation.SourceType.String() + "-" + sourceId)
            }

            source.RemoveAllocationSource(allocation)
            k.SetSubstation(ctx, source)

        case types.ObjectType_reactor:
            source, sourceFound := k.GetReactor(ctx, allocation.SourceId)

            if (!sourceFound){
                sourceId := strconv.FormatUint(allocation.SourceId, 10)
                return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceNotFound, "source (%s) used for allocation not found", allocation.SourceType.String() + "-" + sourceId)
            }

            source.RemoveAllocationSource(allocation)
            k.SetReactor(ctx, source)

        case types.ObjectType_struct:
           //Not Implemented yet

        default:
           return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceTypeMismatch, "Source type (%s) mismatch somehow ", allocation.SourceType.String())
    }


    destinationSubstation, destinationSubstationFound := k.GetSubstation(ctx, allocation.DestinationId)
    if (!destinationSubstationFound){
        destinationId := strconv.FormatUint(allocation.DestinationId, 10)
        return &types.MsgSubstationAllocationDisconnectResponse{}, sdkerrors.Wrapf(types.ErrSubstationNotFound, "destination substation (%s) used for allocation not found", destinationId)
    }

    destinationSubstation.RemoveAllocationDestination(allocation)
    k.SetSubstation(ctx, destinationSubstation)

    k.RemoveAllocation(ctx, msg.AllocationId)

	return &types.MsgSubstationAllocationDisconnectResponse{}, nil
}
