package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
    "cosmossdk.io/math"
)

func (k msgServer) AgreementDurationIncrease(goCtx context.Context, msg *types.MsgAgreementDurationIncrease) (*types.MsgAgreementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := cc.GetPlayerByAddress(msg.Creator)

    agreement := cc.GetAgreement(msg.AgreementId)

    permissionError := agreement.CanUpdate(activePlayer)
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
        return &types.MsgAgreementResponse{}, types.NewPlayerAffordabilityError(activePlayer.GetPlayerId(), "agreement_duration_increase", collateralAmountCoin.String())
    }

    // move the funds from user to provider collateral pool
    errSend := k.bankKeeper.SendCoins(ctx, sourceAcc, agreement.GetProvider().GetCollateralPoolLocation(), collateralAmountCoins)
    if errSend != nil {
        return &types.MsgAgreementResponse{}, errSend
    }

	cc.CommitAll()
	return &types.MsgAgreementResponse{}, nil
}
