package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

/*

message MsgProviderCreate {
  option (cosmos.msg.v1.signer) = "creator";

  string creator            = 1;
  string substationId       = 2;

  cosmos.base.v1beta1.Coin rate  = 3 [ (gogoproto.nullable) = false, (amino.dont_omitempty)   = true, (gogoproto.castrepeated) = "github.com/cosmos/cosmos-sdk/types.Coins"];

  providerAccessPolicy accessPolicy = 4;

  string  providerCancellationPenalty     = 5 [ (cosmos_proto.scalar)  = "cosmos.Dec", (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable)   = false ];
  string  consumerCancellationPenalty     = 6 [(cosmos_proto.scalar)  = "cosmos.Dec",(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable)   = false ];

  uint64 capacityMinimum                  = 7;
  uint64 capacityMaximum                  = 8;
  uint64 durationMinimum                  = 9;
  uint64 durationMaximum                  = 10;
}
*/

func (k msgServer) ProviderCreate(goCtx context.Context, msg *types.MsgProviderCreate) (*types.MsgProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := k.GetPlayerCacheFromAddress(ctx, msg.Creator)

    substation := k.GetSubstationCacheFromId(ctx, msg.SubstationId)


    permissionError := substation.CanCreateAllocations(&activePlayer)
    if (permissionError != nil) {
        return &types.MsgProviderResponse{}, permissionError
    }

    // Create a Provider Object
    provider := types.CreateBaseProvider(msg.Creator, activePlayer.GetPlayerId())

    // TODO Rate Denom whitelist?

    paramErr := provider.SetCapacityRange(msg.CapacityMinimum, msg.CapacityMaximum)
    if paramErr != nil {
        return &types.MsgProviderResponse{}, paramErr
    }

    paramErr = provider.SetDurationRange(msg.DurationMinimum, msg.DurationMaximum )
    if paramErr != nil {
        return &types.MsgProviderResponse{}, paramErr
    }

    paramErr = provider.SetProviderCancellationPenalty(msg.ProviderCancellationPenalty)
    if paramErr != nil {
        return &types.MsgProviderResponse{}, paramErr
    }

    paramErr = provider.SetConsumerCancellationPenalty(msg.ConsumerCancellationPenalty)
    if paramErr != nil {
        return &types.MsgProviderResponse{}, paramErr
    }

    provider.SetAccessPolicy(msg.AccessPolicy)


    // Provider Grid values are OK to leave uninitialized
        // Unset Load is zero
        // Unset CheckpointBlock is zero

    // Pass it to the Keeper
    k.AppendProvider(ctx, provider)


	return &types.MsgProviderResponse{}, nil
}
