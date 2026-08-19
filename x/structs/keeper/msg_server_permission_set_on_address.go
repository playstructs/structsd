package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionSetOnAddress(goCtx context.Context, msg *types.MsgPermissionSetOnAddress) (*types.MsgPermissionResponse, error) {
    emptyResponse := &types.MsgPermissionResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var err error

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
        return  emptyResponse, err
    }

    targetPlayer, err := cc.GetPlayerByAddress(msg.Address)
    if err != nil {
         return  emptyResponse, err
     }

    // This overwrites rather than adds, so the call destroys whatever the target
    // address holds today as well as granting msg.Permissions. Require the
    // caller to hold both. Checking only the incoming bits would let a narrow
    // key rewrite a stronger address of its own player down to its own level --
    // stripping PermAll off a primary address locks the player out of every
    // asset operation permanently, since restoring it needs PermAll.
    targetAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    requiredPermissions := types.Permission(msg.Permissions) | cc.GetPermissions(targetAddressPermissionId)

    permissionErr := targetPlayer.CanRegisterAddressBy(callingPlayer, requiredPermissions)
    if permissionErr != nil {
        return  emptyResponse, permissionErr
    }

    cc.SetPermissions(targetAddressPermissionId, types.Permission(msg.Permissions))

	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
