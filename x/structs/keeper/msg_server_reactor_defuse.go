package keeper

import (
	"context"
    "time"
    //"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k msgServer) ReactorDefuse(goCtx context.Context, msg *types.MsgReactorDefuse) (*types.MsgReactorDefuseResponse, error) {
    emptyResponse := &types.MsgReactorDefuseResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    delegatorAddress, err := cc.resolveReactorDelegator(msg.DelegatorAddress, msg.OwnerId, callingPlayer, types.PermTokenDefuse)
    if err != nil {
        return emptyResponse, err
    }



    valAddr, valErr := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if valErr != nil {
		return emptyResponse, types.NewAddressValidationError(msg.ValidatorAddress, "invalid_validator")
	}

	if !msg.Amount.IsValid() || !msg.Amount.Amount.IsPositive() {
		return emptyResponse, types.NewReactorError("defuse", "invalid_amount")
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return emptyResponse, err
	}

	if msg.Amount.Denom != bondDenom {
		return emptyResponse, types.NewReactorError("defuse", "invalid_denom").WithDenom(msg.Amount.Denom, bondDenom)
	}

	shares, err := k.stakingKeeper.ValidateUnbondAmount(
		ctx, delegatorAddress, valAddr, msg.Amount.Amount,
	)
	if err != nil {
		return emptyResponse, err
	}

	completionTime, undelegatedAmt, err := k.stakingKeeper.Undelegate(ctx, delegatorAddress, valAddr, shares)
	if err != nil {
		return emptyResponse, err
	}

	undelegatedCoin := sdk.NewCoin(msg.Amount.Denom, undelegatedAmt)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			staking.EventTypeUnbond,
			sdk.NewAttribute(staking.AttributeKeyValidator, msg.ValidatorAddress),
			sdk.NewAttribute(staking.AttributeKeyDelegator, delegatorAddress.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, undelegatedCoin.String()),
			sdk.NewAttribute(staking.AttributeKeyCompletionTime, completionTime.Format(time.RFC3339)),
		),
	})

	cc.CommitAll()
	return &types.MsgReactorDefuseResponse{
			CompletionTime: completionTime,
    		Amount:         undelegatedCoin,
    	}, nil
}
