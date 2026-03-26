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

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    // Load the player related to the specified address
    // Normally the address specified should be the PrimaryAddress
    player, err := cc.GetPlayerByAddress(msg.DelegatorAddress)
    if err != nil {
       return emptyResponse, err
    }

    err = player.CanDefuseTokensBy(callingPlayer)
    if err != nil {
       return emptyResponse, err
    }

    delegatorAddress, delegatorAddressErr := sdk.AccAddressFromBech32(msg.DelegatorAddress)
 	if delegatorAddressErr != nil {
 		return emptyResponse, types.NewAddressValidationError(msg.DelegatorAddress, "invalid_delegator")
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
			sdk.NewAttribute(staking.AttributeKeyDelegator, msg.DelegatorAddress),
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
