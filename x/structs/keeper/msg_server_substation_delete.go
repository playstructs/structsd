package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationDelete(goCtx context.Context, msg *types.MsgSubstationDelete) (*types.MsgSubstationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgSubstationDeleteResponse{}, err
    }

    substation := cc.GetSubstation(msg.SubstationId)
    if substation.CheckSubstation() != nil {
        return &types.MsgSubstationDeleteResponse{}, substation.CheckSubstation()
    }

    permissionErr := substation.CanBeDeleteDBy(player)
    if permissionErr != nil {
        return &types.MsgSubstationDeleteResponse{}, permissionErr
    }

    if (msg.MigrationSubstationId != "") {
        migrationSubstation := cc.GetSubstation(msg.MigrationSubstationId)
        if migrationSubstation.CheckSubstation() != nil {
            return &types.MsgSubstationDeleteResponse{}, migrationSubstation.CheckSubstation()
        }

        if migrationSubstation.CanManagePlayerConnections(player) != nil {
            return &types.MsgSubstationDeleteResponse{}, migrationSubstation.CanManagePlayerConnections(player)
        }
        substation.Delete(msg.MigrationSubstationId)
    } else {
        substation.Delete("")
    }

	cc.CommitAll()
	return &types.MsgSubstationDeleteResponse{}, nil
}
