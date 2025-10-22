package keeper

import (
	"context"
    "time"
    //"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k msgServer) ReactorBeginMigration(goCtx context.Context, msg *types.MsgReactorBeginMigration) (*types.MsgReactorBeginMigrationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    valSrcAddr, valSrcErr := sdk.ValAddressFromBech32(msg.ValidatorSrcAddress)
	if valSrcErr != nil {
		return &types.MsgReactorBeginMigrationResponse{}, sdkerrors.Wrapf(types.ErrReactorBeginMigration, "invalid validator address: %s", valSrcErr)
	}

    valDstAddr, valDstErr := sdk.ValAddressFromBech32(msg.ValidatorDstAddress)
	if valDstErr != nil {
		return &types.MsgReactorBeginMigrationResponse{}, sdkerrors.Wrapf(types.ErrReactorBeginMigration, "invalid validator address: %s", valDstErr)
	}

    delegatorAddress, delegatorAddressErr := sdk.AccAddressFromBech32(msg.DelegatorAddress)
 	if delegatorAddressErr != nil {
 		return &types.MsgReactorBeginMigrationResponse{}, sdkerrors.Wrapf(types.ErrReactorBeginMigration, "invalid delegator address: %s", delegatorAddressErr)
 	}

	if !msg.Amount.IsValid() || !msg.Amount.Amount.IsPositive() {
		return &types.MsgReactorBeginMigrationResponse{}, sdkerrors.Wrapf(types.ErrReactorBeginMigration, "invalid delegation amount")
	}

	shares, err := k.stakingKeeper.ValidateUnbondAmount(
		ctx, delegatorAddress, valSrcAddr, msg.Amount.Amount,
	)
	if err != nil {
		return &types.MsgReactorBeginMigrationResponse{}, err
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return &types.MsgReactorBeginMigrationResponse{}, err
	}

	if msg.Amount.Denom != bondDenom {
        return &types.MsgReactorBeginMigrationResponse{}, sdkerrors.Wrapf(types.ErrReactorBeginMigration, "invalid coin denomination: got %s, expected %s", msg.Amount.Denom, bondDenom)
    }

	completionTime, err := k.stakingKeeper.BeginRedelegation(
		ctx, delegatorAddress, valSrcAddr, valDstAddr, shares,
	)
	if err != nil {
		return &types.MsgReactorBeginMigrationResponse{}, err
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

	return &types.MsgReactorBeginMigrationResponse{CompletionTime: completionTime,}, nil
}
