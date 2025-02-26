package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
    "cosmossdk.io/math"
)

func (k msgServer) AgreementDurationIncrease(goCtx context.Context, msg *types.MsgAgreementDurationIncrease) (*types.MsgAgreementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := k.GetPlayerCacheFromAddress(ctx, msg.Creator)

    agreement := k.GetAgreementCacheFromId(ctx, msg.AgreementId)

    permissionError := agreement.CanUpdate(&activePlayer)
    if (permissionError != nil) {
        return &types.MsgAgreementResponse{}, permissionError
    }

    // increase duration by adding more collateral
    paramError := agreement.DurationIncrease(msg.DurationIncrease)
    if paramError != nil {
        return &types.MsgAgreementResponse{}, paramError
    }

    sourceAcc, errParam := sdk.AccAddressFromBech32(activePlayer.GetPrimaryAddress())
    if errParam != nil {
        return &types.MsgAgreementResponse{}, errParam
    }

    // Amount to be sent
    collateralAmountCoin := sdk.NewCoin(agreement.GetProvider().GetRate().Denom, math.NewIntFromUint64(agreement.GetCapacity()).Mul(math.NewIntFromUint64(msg.DurationIncrease).Mul(agreement.GetProvider().GetRate().Amount)))
    collateralAmountCoins := sdk.NewCoins(collateralAmountCoin)


    if !k.bankKeeper.HasBalance(ctx, sourceAcc, collateralAmountCoin) {
        return &types.MsgAgreementResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Player cannot afford the agreement ")
    }

    // move the funds from user to provider collateral pool
    errSend := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sourceAcc, agreement.GetProvider().GetCollateralPoolLocation(), collateralAmountCoins)
    if errSend != nil {
        return &types.MsgAgreementResponse{}, errSend
    }

    agreement.Commit()

	return &types.MsgAgreementResponse{}, nil
}
