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
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Load the player related to the specified address
    // Normally the address specified should be the PrimaryAddress
    player, err := cc.GetPlayerByAddress(msg.DelegatorAddress)
    if err != nil {
       return &types.MsgReactorInfuseResponse{}, err
    }

    // Check if msg.Creator has PermissionAssets on the Address and Account
    err = player.CanBeAdministratedBy(msg.Creator, types.PermissionAssets)
    if err != nil {
       return &types.MsgReactorInfuseResponse{}, err
    }

    delegatorAddress, delegatorAddressErr := sdk.AccAddressFromBech32(msg.DelegatorAddress)
 	if delegatorAddressErr != nil {
 		return &types.MsgReactorInfuseResponse{}, types.NewAddressValidationError(msg.DelegatorAddress, "invalid_delegator")
 	}

    valAddr, valErr := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if valErr != nil {
		return &types.MsgReactorInfuseResponse{}, types.NewAddressValidationError(msg.ValidatorAddress, "invalid_validator")
	}


	if !msg.Amount.IsValid() || !msg.Amount.Amount.IsPositive() {
		return &types.MsgReactorInfuseResponse{}, types.NewReactorError("infuse", "invalid_amount")
	}

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return &types.MsgReactorInfuseResponse{}, err
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return &types.MsgReactorInfuseResponse{}, err
	}

	if msg.Amount.Denom != bondDenom {
		return &types.MsgReactorInfuseResponse{}, types.NewReactorError("infuse", "invalid_denom").WithDenom(msg.Amount.Denom, bondDenom)
	}

	// NOTE: source funds are always unbonded
	newShares, err := k.stakingKeeper.Delegate(ctx, delegatorAddress, msg.Amount.Amount, staking.Unbonded, validator, true)
	if err != nil {
		return &types.MsgReactorInfuseResponse{}, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			staking.EventTypeDelegate,
			sdk.NewAttribute(staking.AttributeKeyValidator, msg.ValidatorAddress),
			sdk.NewAttribute(staking.AttributeKeyDelegator, msg.DelegatorAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
			sdk.NewAttribute(staking.AttributeKeyNewShares, newShares.String()),
		),
	})



	return &types.MsgReactorInfuseResponse{}, nil
}
