package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SubstationCreate(goCtx context.Context, msg *types.MsgSubstationCreate) (*types.MsgSubstationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    player := k.UpsertPlayer(ctx, msg.Creator)
    // check that the account has energy management permissions
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if (playerPermissions&types.AddressPermissionManageEnergy != 0) {
        return &types.MsgSubstationCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageEnergy, "Calling address (%s) has no Energy Management permissions ", msg.Creator)
    }



    substation := k.AppendSubstation(ctx, msg.PlayerConnectionAllocation, player)

    // Now let's get the player some power
    if (player.SubstationId == 0) {
        // Connect Player to Substation
        k.SubstationIncrementConnectedPlayerLoad(ctx, substation.Id, 1)
        player.SetSubstation(substation.Id)
        k.SetPlayer(ctx, player)
    }


	return &types.MsgSubstationCreateResponse{SubstationId: substation.Id}, nil
}
