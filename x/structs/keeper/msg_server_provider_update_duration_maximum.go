package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) ProviderUpdateDurationMaximum(goCtx context.Context, msg *types.MsgProviderUpdateDurationMaximum) (*types.MsgProviderResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := k.GetPlayerCacheFromAddress(ctx, msg.Creator)

    provider := k.GetProviderCacheFromId(ctx, msg.ProviderId)

    permissionError := provider.CanUpdate(&activePlayer)
    if (permissionError != nil) {
        return &types.MsgProviderResponse{}, permissionError
    }

    paramErr := provider.SetDurationMaximum(msg.NewMaximumDuration)
    if paramErr != nil {
        return &types.MsgProviderResponse{}, paramErr
    }

    provider.Commit()

	return &types.MsgProviderResponse{}, nil
}
