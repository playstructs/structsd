package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PermissionGrantOnAddress(goCtx context.Context, msg *types.MsgPermissionGrantOnAddress) (*types.MsgPermissionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)


    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    var err error

    if msg.Permissions == 0 {
        return &types.MsgPermissionResponse{}, types.NewParameterValidationError("permissions", 0, "below_minimum").WithRange(1, 0)
    }

    callingPlayer, _ := cc.GetPlayerByAddress(msg.Creator)
    err = callingPlayer.CheckPlayer()
    if err != nil {
        return  &types.MsgPermissionResponse{}, err
    }

    targetPlayer, _ := cc.GetPlayerByAddress(msg.Address)
    err = targetPlayer.CheckPlayer()
    if err != nil {
         return  &types.MsgPermissionResponse{}, err
    }

    permissionErr := targetPlayer.CanRegisterAddressBy(callingPlayer, types.Permission(msg.Permissions))
    if permissionErr != nil {
        return  &types.MsgPermissionResponse{}, permissionErr
    }

    targetAddressPermissionId := GetAddressPermissionIDBytes(msg.Address)
    cc.PermissionAdd(targetAddressPermissionId, types.Permission(msg.Permissions))

	cc.CommitAll()
	return &types.MsgPermissionResponse{}, nil
}
