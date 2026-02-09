package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationDelete(goCtx context.Context, msg *types.MsgSubstationDelete) (*types.MsgSubstationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgSubstationDeleteResponse{}, err
    }


    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.GetPlayerId())
	// check that the player has reactor permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionDelete)) {
        return &types.MsgSubstationDeleteResponse{}, types.NewPermissionError("player", player.GetPlayerId(), "substation", msg.SubstationId, uint64(types.PermissionDelete), "substation_delete")
    }


    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
        return &types.MsgSubstationDeleteResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "energy_management")
    }



	k.RemoveSubstation(ctx, msg.SubstationId, msg.MigrationSubstationId)

	return &types.MsgSubstationDeleteResponse{}, nil
}
