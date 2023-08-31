package keeper

import (

    "context"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)


func (k Keeper) GetBlockHeight(goCtx context.Context, req *types.QueryBlockHeight) (*types.QueryBlockHeightResponse, error)  {
    ctx := sdk.UnwrapSDKContext(goCtx)
    return &types.QueryBlockHeightResponse{ BlockHeight: uint64(ctx.BlockHeight()) }, nil
}