package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"cosmossdk.io/math"
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

    substation := k.GetSubstationCacheFromId(ctx, msg.SubstationId)

    permissionError := substation.CanCreateAllocations(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgProviderResponse{}, permissionError
    }

    // TODO Rate Denom whitelist?

    // Capacity Minimum < Capacity Maximum
    if msg.CapacityMinimum > msg.CapacityMaximum {
        return &types.MsgProviderResponse{}, sdkerrors.Wrapf(types.ErrInvalidParameters, "Minimum Capacity (%d) cannot be larger than Maximum Capacity (%d)", msg.CapacityMinimum, msg.CapacityMaximum)
    }

    // Duration Minimum < Duration Maximum
    if msg.DurationMinimum > msg.DurationMaximum {
        return &types.MsgProviderResponse{}, sdkerrors.Wrapf(types.ErrInvalidParameters, "Minimum Duration (%d) cannot be larger than Maximum Duration (%d)", msg.DurationMinimum, msg.DurationMaximum)
    }

    one, _ := math.LegacyNewDecFromStr("1")

    // 1 <= Provider Cancellation Policy => 0
    if msg.ProviderCancellationPenalty.GTE(math.LegacyZeroDec()) && msg.ProviderCancellationPenalty.LTE(one) {
        return &types.MsgProviderResponse{}, sdkerrors.Wrapf(types.ErrInvalidParameters, "Provider Cancellation Penalty (%f) must be between 1 and 0", msg.ProviderCancellationPenalty)
    }


    // 1 <= Consumer Cancellation Policy => 0
    if msg.ConsumerCancellationPenalty.GTE(math.LegacyZeroDec()) && msg.ConsumerCancellationPenalty.LTE(one) {
        return &types.MsgProviderResponse{}, sdkerrors.Wrapf(types.ErrInvalidParameters, "Provider Cancellation Penalty (%f) must be between 1 and 0", msg.ConsumerCancellationPenalty)
    }


	return &types.MsgProviderResponse{}, nil
}
