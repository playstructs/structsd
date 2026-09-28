package keeper

import (
	"context"
    //"time"
    //"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k msgServer) ReactorInfuse(goCtx context.Context, msg *types.MsgReactorInfuse) (*types.MsgReactorInfuseResponse, error) {
    emptyResponse := &types.MsgReactorInfuseResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    delegatorAddress, err := cc.resolveReactorDelegator(msg.DelegatorAddress, msg.OwnerId, callingPlayer, types.PermTokenInfuse)
    if err != nil {
        return emptyResponse, err
    }



    valAddr, valErr := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if valErr != nil {
		return emptyResponse, types.NewAddressValidationError(msg.ValidatorAddress, "invalid_validator")
	}


	if !msg.Amount.IsValid() || !msg.Amount.Amount.IsPositive() {
		return emptyResponse, types.NewReactorError("infuse", "invalid_amount")
	}

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return emptyResponse, err
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return emptyResponse, err
	}

	if msg.Amount.Denom != bondDenom {
		return emptyResponse, types.NewReactorError("infuse", "invalid_denom").WithDenom(msg.Amount.Denom, bondDenom)
	}

	// NOTE: source funds are always unbonded
	newShares, err := k.stakingKeeper.Delegate(ctx, delegatorAddress, msg.Amount.Amount, staking.Unbonded, validator, true)
	if err != nil {
		return emptyResponse, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			staking.EventTypeDelegate,
			sdk.NewAttribute(staking.AttributeKeyValidator, msg.ValidatorAddress),
			sdk.NewAttribute(staking.AttributeKeyDelegator, delegatorAddress.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
			sdk.NewAttribute(staking.AttributeKeyNewShares, newShares.String()),
		),
	})



	cc.CommitAll()
	return &types.MsgReactorInfuseResponse{}, nil
}
