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

    // Load Player

    // Check Permissions of Creator

    // Check Substation Permissions

    // TODO Rate Denom whitelist?

    // Capacity Minimum < Capacity Maximum

    // Duration Minimum < Duration Maximum

    // 1 <= Provider Cancellation Policy => 0

    // 1 <= Consumer Cancellation Policy => 0


	return &types.MsgProviderResponse{}, nil
}
