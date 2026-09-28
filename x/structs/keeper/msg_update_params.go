package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
    next := req.Params.WithReactorDefaults()
    if err := next.Validate(); err != nil { return nil, err }
    current := k.GetParams(ctx).WithReactorDefaults()
    if next.DiversityMaxBonusBps != current.DiversityMaxBonusBps || next.DiversityTargetExtraReactors != current.DiversityTargetExtraReactors {
        return nil, errorsmod.Wrap(types.ErrInvalidParameters, "diversity pricing changes require an upgrade")
    }
	if err := k.SetParams(ctx, next); err != nil {
		return nil, err
	}

	if err := k.EmitReactorStakeCap(ctx); err != nil { return nil, err }
	return &types.MsgUpdateParamsResponse{}, nil
}
