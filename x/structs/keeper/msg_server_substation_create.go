package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

/*
message MsgSubstationCreate {
  string creator             = 1;
  string permissionsOverride = 2;
  string connect             = 3;
}

message MsgSubstationCreateResponse {}
*/

func (k msgServer) SubstationCreate(goCtx context.Context, msg *types.MsgSubstationCreate) (*types.MsgSubstationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	substation := types.CreateEmptySubstation()

    k.AppendSubstation(ctx, substation)

	return &types.MsgSubstationCreateResponse{}, nil
}
