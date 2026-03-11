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

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return  emptyResponse, err
    }

    targetPlayer, err := cc.GetPlayerByAddress(msg.Address)
    if err != nil {
         return  emptyResponse, err
     }

    permissionErr := targetPlayer.CanRegisterAddressBy(callingPlayer, types.Permission(msg.Permissions))
    if permissionErr != nil {
        return  emptyResponse, permissionErr
    }

    targetAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    cc.SetPermissions(targetAddressPermissionId, types.Permission(msg.Permissions))

	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
