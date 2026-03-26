package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationDelete(goCtx context.Context, msg *types.MsgSubstationDelete) (*types.MsgSubstationDeleteResponse, error) {
    emptyResponse := &types.MsgSubstationDeleteResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    substation := cc.GetSubstation(msg.SubstationId)
    if substation.CheckSubstation() != nil {
        return emptyResponse, substation.CheckSubstation()
    }

    permissionErr := substation.CanBeDeleteDBy(callingPlayer)
    if permissionErr != nil {
        return emptyResponse, permissionErr
    }

    if (msg.MigrationSubstationId != "") {
        migrationSubstation := cc.GetSubstation(msg.MigrationSubstationId)
        if migrationSubstation.CheckSubstation() != nil {
            return emptyResponse, migrationSubstation.CheckSubstation()
        }

        substationPermissionErr := migrationSubstation.CanManageConnectionsBy(callingPlayer)
        if substationPermissionErr != nil {
            return emptyResponse, substationPermissionErr
        }
        substation.Delete(msg.MigrationSubstationId)
    } else {
        substation.Delete("")
    }

	cc.CommitAll()
	return &types.MsgSubstationDeleteResponse{}, nil
}
