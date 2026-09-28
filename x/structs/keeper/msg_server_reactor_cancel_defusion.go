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
    emptyResponse := &types.MsgReactorCancelDefusionResponse{}
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
		return emptyResponse, types.NewReactorError("cancel_defusion", "invalid_address").WithAddress(msg.ValidatorAddress, "validator")
	}

	if !msg.Amount.IsValid() || !msg.Amount.Amount.IsPositive() {
		return emptyResponse, types.NewReactorError("cancel_defusion", "invalid_amount")
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return emptyResponse, err
	}

	if msg.Amount.Denom != bondDenom {
		return emptyResponse, types.NewReactorError("cancel_defusion", "invalid_denom").WithDenom(msg.Amount.Denom, bondDenom)
	}

	if msg.CreationHeight <= 0 {
		return emptyResponse, types.NewReactorError("cancel_defusion", "invalid_height")
	}

	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return emptyResponse, err
	}

	// In some situations, the exchange rate becomes invalid, e.g. if
	// Validator loses all tokens due to slashing. In this case,
	// make all future delegations invalid.
	if validator.InvalidExRate() {
		return emptyResponse, staking.ErrDelegatorShareExRateInvalid
	}

	if validator.IsJailed() {
		return emptyResponse, staking.ErrValidatorJailed
	}

	ubd, err := k.stakingKeeper.GetUnbondingDelegation(ctx, delegatorAddress, valAddr)
	if err != nil {
		return emptyResponse, types.NewObjectNotFoundError("unbonding_delegation", msg.DelegatorAddress).WithContext("validator: " + msg.ValidatorAddress)
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
        return emptyResponse, types.NewReactorError("cancel_defusion", "entry_not_found").WithHeight(msg.CreationHeight)
    }

	if unbondEntry.Balance.LT(msg.Amount.Amount) {
		return emptyResponse, types.NewReactorError("cancel_defusion", "balance_exceeded")
	}

	if unbondEntry.CompletionTime.Before(ctx.BlockTime()) {
		return emptyResponse, types.NewReactorError("cancel_defusion", "already_processed")
	}

	// delegate back the unbonding delegation amount to the validator
	_, err = k.stakingKeeper.Delegate(ctx, delegatorAddress, msg.Amount.Amount, staking.Unbonding, validator, false)
	if err != nil {
		return emptyResponse, err
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
		return emptyResponse, err
	}

	// Delegate's hooks saw the old unbonding balance. Refresh only after the
	// updated UBD is stored, using a fresh context as staking hooks do.
	if err := k.ReactorUpdatePlayerInfusion(ctx, delegatorAddress, valAddr); err != nil {
        return nil, err
    }

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			staking.EventTypeCancelUnbondingDelegation,
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Amount.String()),
			sdk.NewAttribute(staking.AttributeKeyValidator, msg.ValidatorAddress),
			sdk.NewAttribute(staking.AttributeKeyDelegator, delegatorAddress.String()),
			sdk.NewAttribute(staking.AttributeKeyCreationHeight, strconv.FormatInt(msg.CreationHeight, 10)),
		),
	)


//


	cc.CommitAll()
	return &types.MsgReactorCancelDefusionResponse{}, nil
}
