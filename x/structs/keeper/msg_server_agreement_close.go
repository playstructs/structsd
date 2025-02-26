package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) AgreementClose(goCtx context.Context, msg *types.MsgAgreementClose) (*types.MsgAgreementResponse, error) {
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

    // Checkpoint
    agreement.GetProvider().Checkpoint()
    remainingCollateral, errorParam := agreement.CloseAndCommit()
    if (errorParam != nil) {
        return &types.MsgAgreementResponse{}, errorParam
    }

    sourceAcc, errParam := sdk.AccAddressFromBech32(activePlayer.GetPrimaryAddress())
    if errParam != nil {
        return &types.MsgAgreementResponse{}, errParam
    }
    k.bankKeeper.SendCoinsFromModuleToAccount(ctx, agreement.GetProvider().GetCollateralPoolLocation(), sourceAcc, remainingCollateral)


	return &types.MsgAgreementResponse{}, nil
}
