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

	substation := types.CreateEmptySubstation()
	substation.SetCreator(msg.Creator)


	// TODO Have this build the player object and pass that instead
	// this will enforce the check that a player actually exists before it's
	// given ownership over something.
	substation.SetOwner(msg.Owner)


	substation.SetPlayerConnectionAllocation(msg.PlayerConnectionAllocation)

    newSubstationId := k.AppendSubstation(ctx, substation)

	return &types.MsgSubstationCreateResponse{SubstationId: newSubstationId}, nil
}
