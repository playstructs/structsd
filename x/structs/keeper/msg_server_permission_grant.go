package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) PermissionGrant(goCtx context.Context, msg *types.MsgPermissionGrant) (*types.MsgPermissionGrantResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    player, playerFound := k.GetPlayerFromIndex(ctx, k.GetPlayerIndexFromAddress(ctx, msg.Creator), false)

/*    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)


    // Make sure the address calling this has Associate permissions
    if (k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionManageGuild))) {
        return &types.MsgPermissionGrantResponse{}, sdkerrors.Wrapf(types.ErrPermissionAssociation, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }

    // The calling address must have a minimum of the same permission level
    if (k.PermissionHasAll(ctx, addressPermissionId, types.Permission(msg.Permissions))) {
        return &types.MsgPermissionGrantResponse{}, sdkerrors.Wrapf(types.ErrPermissionAssociation, "Calling address (%s) does not have permissions needed to allow address association of higher functionality ", msg.Creator)
    }
*/
    // TODO something
    _ = player
    _ = playerFound

	return &types.MsgPermissionGrantResponse{}, nil
}
