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
    activePlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    provider, providerErr := cc.GetExistingProvider(msg.ProviderId)
    if providerErr != nil {
        return emptyResponse, providerErr
    }

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

    // Every agreement expiring at one height is torn down in that block's
    // EndBlocker, which has no gas meter, so a height holds only so many. The
    // consumer picks the duration and therefore the height; a full one means
    // choosing another. See AgreementExpirationBucketCap.
    //
    // Checked here with the rest of the parameter validation, ahead of the
    // collateral transfer below, so a refusal never has to unwind a payment.
    if !k.AgreementExpirationHeightHasRoomFor(ctx, uint64(ctx.BlockHeight()) + msg.Duration, "") {
        return emptyResponse, types.NewParameterValidationError("duration", msg.Duration, "expiration_height_full").WithRange(0, types.AgreementExpirationBucketCap)
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
    cc.SetPermissions(allocationPermissionId, types.PermAllocationConnection)

    // Build the Agreement through context.
    //
    // Service starts in this block, not the next one, because the load increase
    // below takes effect immediately and Checkpoint() bills aggregate load from
    // the checkpoint block — which the Checkpoint above just set to this height.
    // Starting a block later billed the provider for one block of service the
    // consumer never received, leaving the collateral pool short by that much and
    // tripping the provider-collateral-solvency invariant.
    startBlock := uint64(ctx.BlockHeight())
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
    cc.SetPermissions(agreementPermissionId, types.PermAgreementAll)

    provider.AgreementLoadIncrease(msg.Capacity)

	cc.CommitAll()
	return &types.MsgAgreementResponse{}, nil
}
