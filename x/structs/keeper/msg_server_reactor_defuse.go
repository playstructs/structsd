package keeper

import (
	"context"
    //"time"
    //"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	//staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k msgServer) ReactorDefuse(goCtx context.Context, msg *types.MsgReactorDefuse) (*types.MsgReactorDefuseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


	return &types.MsgReactorDefuseResponse{}, nil
}
