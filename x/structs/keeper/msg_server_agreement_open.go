package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
    emptyResponse := &types.MsgAgreementResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    provider := cc.GetProvider(msg.ProviderId)

    permissionError := provider.CanOpenAgreement(activePlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    // Are agreement details valid?
    // Does the substation have enough capacity?
    paramError := provider.AgreementVerify(msg.Capacity, msg.Duration)
    if (paramError != nil) {
        return emptyResponse, paramError
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
        return emptyResponse, errParam
    }

    if !k.bankKeeper.HasBalance(ctx, sourceAcc, collateralAmountCoin) {
        return emptyResponse, types.NewPlayerAffordabilityError(activePlayer.GetPlayerId(), "agreement_open", collateralAmountCoin.String())
    }

    // move the funds from user to provider collateral pool
    errSend := k.bankKeeper.SendCoins(ctx, sourceAcc, provider.GetCollateralPoolLocation(), collateralAmountCoins)
    if errSend != nil {
        return emptyResponse, errSend
    }

    checkpointError := provider.Checkpoint()
    if checkpointError != nil {
        return emptyResponse, checkpointError
    }

    // Create the allocation through context
    allocation, allocationErr := cc.NewAllocation(
        types.AllocationType_providerAgreement,
        provider.GetSubstationId(),
        "",
        msg.Creator,
        activePlayer.GetPlayerId(),
        msg.Capacity,
    )
    if allocationErr != nil {
        return emptyResponse, allocationErr
    }

    allocationPermissionId := GetObjectPermissionIDBytes(allocation.ID(), activePlayer.ID())
    cc.k.SetPermissions(allocationPermissionId, types.PermAllocationConnection)

    // Build the Agreement through context
    startBlock := uint64(ctx.BlockHeight()) + 1
    endBlock := startBlock + msg.Duration

    agreementRecord := types.CreateBaseAgreement(
        msg.Creator,
        activePlayer.GetPlayerId(),
        msg.ProviderId,
        msg.Capacity,
        startBlock,
        endBlock,
        allocation.GetAllocation().Id,
    )
    agreementRecord.Id = GetObjectID(types.ObjectType_agreement, allocation.GetAllocation().Index)
    agreement := cc.NewAgreement(agreementRecord)

    agreementPermissionId := GetObjectPermissionIDBytes(agreement.ID(), activePlayer.ID())
    cc.k.SetPermissions(agreementPermissionId, types.PermAgreementAll)

    provider.AgreementLoadIncrease(msg.Capacity)

	cc.CommitAll()
	return &types.MsgAgreementResponse{}, nil
}
