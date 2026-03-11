package keeper

import (
	"context"
    "time"
    //"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k msgServer) ReactorBeginMigration(goCtx context.Context, msg *types.MsgReactorBeginMigration) (*types.MsgReactorBeginMigrationResponse, error) {
    emptyResponse := &types.MsgReactorBeginMigrationResponse{}
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

    err = player.CanMigrateTokensBy(callingPlayer)
    if err != nil {
       return emptyResponse, err
    }

    valSrcAddr, valSrcErr := sdk.ValAddressFromBech32(msg.ValidatorSrcAddress)
	if valSrcErr != nil {
		return emptyResponse, types.NewAddressValidationError(msg.ValidatorSrcAddress, "invalid_validator")
	}

    valDstAddr, valDstErr := sdk.ValAddressFromBech32(msg.ValidatorDstAddress)
	if valDstErr != nil {
		return emptyResponse, types.NewAddressValidationError(msg.ValidatorDstAddress, "invalid_validator")
	}

    delegatorAddress, delegatorAddressErr := sdk.AccAddressFromBech32(msg.DelegatorAddress)
 	if delegatorAddressErr != nil {
 		return emptyResponse, types.NewAddressValidationError(msg.DelegatorAddress, "invalid_delegator")
 	}

	if !msg.Amount.IsValid() || !msg.Amount.Amount.IsPositive() {
		return emptyResponse, types.NewReactorError("begin_migration", "invalid_amount")
	}

	shares, err := k.stakingKeeper.ValidateUnbondAmount(
		ctx, delegatorAddress, valSrcAddr, msg.Amount.Amount,
	)
	if err != nil {
		return emptyResponse, err
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return emptyResponse, err
	}

	if msg.Amount.Denom != bondDenom {
        return emptyResponse, types.NewReactorError("begin_migration", "invalid_denom").WithDenom(msg.Amount.Denom, bondDenom)
    }

	completionTime, err := k.stakingKeeper.BeginRedelegation(
		ctx, delegatorAddress, valSrcAddr, valDstAddr, shares,
	)
	if err != nil {
		return emptyResponse, err
	}



	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			staking.EventTypeRedelegate,
			sdk.NewAttribute(staking.AttributeKeySrcValidator, msg.ValidatorSrcAddress),
			sdk.NewAttribute(staking.AttributeKeyDstValidator, msg.ValidatorDstAddress),
			sdk.NewAttribute(staking.AttributeKeyDelegator, msg.DelegatorAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
			sdk.NewAttribute(staking.AttributeKeyCompletionTime, completionTime.Format(time.RFC3339)),
		),
	})

	cc.CommitAll()
	return &types.MsgReactorBeginMigrationResponse{CompletionTime: completionTime,}, nil
}
