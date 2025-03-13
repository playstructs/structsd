package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
    "cosmossdk.io/math"
)

/*
message MsgAgreementOpen {
  option (cosmos.msg.v1.signer) = "creator";

  string creator            = 1;
  string playerId           = 2;
  string providerId         = 3;
  uint64 duration           = 4;
  uint64 capacity           = 5;
}
*/

func (k msgServer) AgreementOpen(goCtx context.Context, msg *types.MsgAgreementOpen) (*types.MsgAgreementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := k.GetPlayerCacheFromAddress(ctx, msg.Creator)

    provider := k.GetProviderCacheFromId(ctx, msg.ProviderId)

    permissionError := provider.CanOpenAgreement(&activePlayer)
    if (permissionError != nil) {
        return &types.MsgAgreementResponse{}, permissionError
    }

    // Are agreement details valid?
    // Does the substation have enough capacity?
    paramError := provider.AgreementVerify(msg.Capacity, msg.Duration)
    if (paramError != nil) {
        return &types.MsgAgreementResponse{}, paramError
    }

    // Does the activePlayer have enough for the collateral
    duration := math.NewIntFromUint64(msg.Duration)
    capacity := math.NewIntFromUint64(msg.Capacity)
    collateralAmount := duration.Mul(capacity).Mul(provider.GetRate().Amount)
    //balanceError := activePlayer.CanAffordAgreement(collateralAmount, provider.GetRate().Denom)
    collateralAmountCoin := sdk.NewCoin(provider.GetRate().Denom, collateralAmount)
    collateralAmountCoins := sdk.NewCoins(collateralAmountCoin)
    sourceAcc, errParam := sdk.AccAddressFromBech32(activePlayer.GetPrimaryAddress())
    if errParam != nil {
        return &types.MsgAgreementResponse{}, errParam
    }

    if !k.bankKeeper.HasBalance(ctx, sourceAcc, collateralAmountCoin) {
        return &types.MsgAgreementResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Player cannot afford the agreement ")
    }

    // move the funds from user to provider collateral pool
    errSend := k.bankKeeper.SendCoins(ctx, sourceAcc, provider.GetCollateralPoolLocation(), collateralAmountCoins)
    if errSend != nil {
        return &types.MsgAgreementResponse{}, errSend
    }

    // Create the allocation
    allocation := types.CreateAllocationStub(types.AllocationType_providerAgreement, provider.GetSubstationId(), msg.Creator, activePlayer.GetPlayerId())
    allocation, _ , _ = k.AppendAllocation(ctx, allocation, msg.Capacity)

    // Build the Agreement range
    startBlock := uint64(ctx.BlockHeight()) + uint64(1)
    endBlock := startBlock + msg.Duration

    agreement := types.CreateBaseAgreement(msg.Creator, activePlayer.GetPlayerId(), msg.ProviderId, msg.Capacity, startBlock, endBlock, allocation.Id)
    // Append the Agreement using the Allocations Id Index
    agreement.Id = GetObjectID(types.ObjectType_agreement, allocation.Index)
    k.AppendAgreement(ctx, agreement)

    checkpointError := provider.Checkpoint()
    if checkpointError != nil {
        return &types.MsgAgreementResponse{}, checkpointError
    }
    provider.AgreementLoadIncrease(msg.Capacity)
    provider.Commit()

	return &types.MsgAgreementResponse{}, nil
}
