package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationDelete(goCtx context.Context, msg *types.MsgSubstationDelete) (*types.MsgSubstationDeleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator))
    if (!playerFound) {
        return &types.MsgSubstationDeleteResponse{}, types.NewPlayerRequiredError(msg.Creator, "substation_delete")
    }


    substationObjectPermissionId := GetObjectPermissionIDBytes(msg.SubstationId, player.Id)
	// check that the player has reactor permissions
    if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionDelete)) {
        return &types.MsgSubstationDeleteResponse{}, types.NewPermissionError("player", player.Id, "substation", msg.SubstationId, uint64(types.PermissionDelete), "substation_delete")
    }


    // check that the account has energy management permissions
    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssets)) {
        return &types.MsgSubstationDeleteResponse{}, types.NewPermissionError("address", msg.Creator, "", "", uint64(types.PermissionAssets), "energy_management")
    }



	k.RemoveSubstation(ctx, msg.SubstationId, msg.MigrationSubstationId)

	return &types.MsgSubstationDeleteResponse{}, nil
}
