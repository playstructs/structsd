package keeper

import (
	"context"
    //"time"
    "strconv"
    //"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k msgServer) ReactorCancelDefusion(goCtx context.Context, msg *types.MsgReactorCancelDefusion) (*types.MsgReactorCancelDefusionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Load the player related to the specified address
    // Normally the address specified should be the PrimaryAddress
    player, err := k.GetPlayerCacheFromAddress(ctx, msg.DelegatorAddress)
    if err != nil {
       return &types.MsgReactorCancelDefusionResponse{}, err
    }

    // Check if msg.Creator has PermissionAssets on the Address and Account
    err = player.CanBeAdministratedBy(msg.Creator, types.PermissionAssets)
    if err != nil {
       return &types.MsgReactorCancelDefusionResponse{}, err
    }


    delegatorAddress, delegatorAddressErr := sdk.AccAddressFromBech32(msg.DelegatorAddress)
 	if delegatorAddressErr != nil {
 		return &types.MsgReactorCancelDefusionResponse{}, types.NewReactorError("cancel_defusion", "invalid_address").WithAddress(msg.DelegatorAddress, "delegator")
 	}

    valAddr, valErr := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if valErr != nil {
		return &types.MsgReactorCancelDefusionResponse{}, types.NewReactorError("cancel_defusion", "invalid_address").WithAddress(msg.ValidatorAddress, "validator")
	}

	if !msg.Amount.IsValid() || !msg.Amount.Amount.IsPositive() {
		return &types.MsgReactorCancelDefusionResponse{}, types.NewReactorError("cancel_defusion", "invalid_amount")
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return &types.MsgReactorCancelDefusionResponse{}, err
	}

	if msg.Amount.Denom != bondDenom {
		return &types.MsgReactorCancelDefusionResponse{}, types.NewReactorError("cancel_defusion", "invalid_denom").WithDenom(msg.Amount.Denom, bondDenom)
	}

	if msg.CreationHeight <= 0 {
		return &types.MsgReactorCancelDefusionResponse{}, types.NewReactorError("cancel_defusion", "invalid_height")
	}

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return &types.MsgReactorCancelDefusionResponse{}, err
	}

	// In some situations, the exchange rate becomes invalid, e.g. if
	// Validator loses all tokens due to slashing. In this case,
	// make all future delegations invalid.
	if validator.InvalidExRate() {
		return &types.MsgReactorCancelDefusionResponse{}, staking.ErrDelegatorShareExRateInvalid
	}

	if validator.IsJailed() {
		return &types.MsgReactorCancelDefusionResponse{}, staking.ErrValidatorJailed
	}

	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, delegatorAddress, valAddr)
	if err != nil {
		return &types.MsgReactorCancelDefusionResponse{}, types.NewObjectNotFoundError("unbonding_delegation", msg.DelegatorAddress).WithContext("validator: " + msg.ValidatorAddress)
	}

    var (
        unbondEntry      staking.UnbondingDelegationEntry
        unbondEntryIndex int64 = -1
    )

    for i, entry := range ubd.Entries {
        if entry.CreationHeight == msg.CreationHeight {
            unbondEntry = entry
            unbondEntryIndex = int64(i)
            break
        }
    }
    if unbondEntryIndex == -1 {
        return &types.MsgReactorCancelDefusionResponse{}, types.NewReactorError("cancel_defusion", "entry_not_found").WithHeight(msg.CreationHeight)
    }

	if unbondEntry.Balance.LT(msg.Amount.Amount) {
		return &types.MsgReactorCancelDefusionResponse{}, types.NewReactorError("cancel_defusion", "balance_exceeded")
	}

	if unbondEntry.CompletionTime.Before(ctx.BlockTime()) {
		return &types.MsgReactorCancelDefusionResponse{}, types.NewReactorError("cancel_defusion", "already_processed")
	}

	// delegate back the unbonding delegation amount to the validator
	_, err = k.stakingKeeper.Delegate(ctx, delegatorAddress, msg.Amount.Amount, staking.Unbonding, validator, false)
	if err != nil {
		return &types.MsgReactorCancelDefusionResponse{}, err
	}

	amount := unbondEntry.Balance.Sub(msg.Amount.Amount)
	if amount.IsZero() {
		ubd.RemoveEntry(unbondEntryIndex)
	} else {
		// update the unbondingDelegationEntryBalance and InitialBalance for ubd entry
		unbondEntry.Balance = amount
		unbondEntry.InitialBalance = unbondEntry.InitialBalance.Sub(msg.Amount.Amount)
		ubd.Entries[unbondEntryIndex] = unbondEntry
	}

	// set the unbonding delegation or remove it if there are no more entries
	if len(ubd.Entries) == 0 {
		err = k.stakingKeeper.RemoveUnbondingDelegation(ctx, ubd)
	} else {
		err = k.stakingKeeper.SetUnbondingDelegation(ctx, ubd)
	}

	if err != nil {
		return &types.MsgReactorCancelDefusionResponse{}, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			staking.EventTypeCancelUnbondingDelegation,
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
			sdk.NewAttribute(staking.AttributeKeyValidator, msg.ValidatorAddress),
			sdk.NewAttribute(staking.AttributeKeyDelegator, msg.DelegatorAddress),
			sdk.NewAttribute(staking.AttributeKeyCreationHeight, strconv.FormatInt(msg.CreationHeight, 10)),
		),
	)


//


	return &types.MsgReactorCancelDefusionResponse{}, nil
}
