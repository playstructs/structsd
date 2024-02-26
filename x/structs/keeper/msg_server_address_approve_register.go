package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) AddressApproveRegister(goCtx context.Context, msg *types.MsgAddressApproveRegister) (*types.MsgAddressRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)

    // Make sure the address calling this has Associate permissions
    if (k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionManageGuild))) {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionAssociation, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }

    // The calling address must have a minimum of the same permission level
    if (k.PermissionHasAll(ctx, addressPermissionId, types.Permission(msg.Permissions))) {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrPermissionAssociation, "Calling address (%s) does not have permissions needed to allow address association of higher functionality ", msg.Creator)
    }


    if (playerFound) {
        if (msg.Approve) {
            // TODO permission checking to see if this specific account has the ability to grant these permissions

            k.AddressApproveRegisterRequest(ctx, player, msg.Address, desiredPermissions)
        } else {
            k.AddressDenyRegisterRequest(ctx, player, msg.Address)
        }
    }

	return &types.MsgAddressRegisterResponse{}, nil
}
