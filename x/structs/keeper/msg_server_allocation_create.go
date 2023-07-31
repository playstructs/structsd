package keeper

import (
	"context"
	"structs/x/structs/types"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)


func (k msgServer) AllocationCreate(goCtx context.Context, msg *types.MsgAllocationCreate) (*types.MsgAllocationCreateResponse, error) {
    /*
     * This section is a little repetitive due to the fact that I can't
     * just have a generic source variable that can switch between types
     *
     */
    switch msg.SourceType {
        case types.ObjectType_substation:
            return k.SubstationAllocationCreate(goCtx, &types.MsgSubstationAllocationCreate{
                  Creator: msg.Creator,
                  Controller: msg.Controller,
                  SourceId: msg.SourceId,
                  Power: msg.Power,
            })

        case types.ObjectType_reactor:
            return k.ReactorAllocationCreate(goCtx, &types.MsgReactorAllocationCreate{
                  Creator: msg.Creator,
                  Controller: msg.Controller,
                  SourceId: msg.SourceId,
                  Power: msg.Power,
            })

        case types.ObjectType_struct:
           //Not Implemented yet

        default:
           return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceTypeMismatch, "Source type (%s) mismatch somehow ", msg.SourceType.String())
    }

    return &types.MsgAllocationCreateResponse{}, sdkerrors.Wrapf(types.ErrAllocationSourceTypeMismatch, "Source type (%s) mismatch somehow ", msg.SourceType.String())

}
