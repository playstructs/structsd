package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationCreate(goCtx context.Context, msg *types.MsgSubstationCreate) (*types.MsgSubstationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    player := k.UpsertPlayer(ctx, msg.Creator)

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
