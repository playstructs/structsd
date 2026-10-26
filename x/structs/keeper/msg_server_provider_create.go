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
    emptyResponse := &types.MsgProviderResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, lookupErr := cc.GetSigningPlayer(msg.Creator)
    if lookupErr != nil {
        return emptyResponse, lookupErr
    }

    substation, substationErr := cc.GetExistingSubstation(msg.SubstationId)
    if substationErr != nil {
        return emptyResponse, substationErr
    }

    permissionError := substation.CanAllocateAsSourceBy(activePlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    ownerId := msg.OwnerId
    if ownerId == "" {
        ownerId = activePlayer.ID()
    }
    owner, ownerErr := cc.GetExistingAssetOwner(ownerId)
    if ownerErr != nil {
        return emptyResponse, ownerErr
    }
    if _, guildOwned := owner.(*GuildCache); guildOwned {
        if err := cc.PermissionCheck(owner, activePlayer, types.PermSourceAllocation); err != nil {
            return emptyResponse, err
        }
    }

    // Create a Provider Object
    provider := types.CreateBaseProvider(msg.Creator, owner.ID())

    provider.SetSubstationId(msg.SubstationId)

    // TODO Rate Denom whitelist?
    if msg.Rate.Denom == "" {
        return emptyResponse, types.NewObjectNotFoundError("denom", "")
    }

    paramErr := provider.SetRate(msg.Rate)
    if paramErr != nil {
        return emptyResponse, paramErr
    }

    paramErr = provider.SetCapacityRange(msg.CapacityMinimum, msg.CapacityMaximum)
    if paramErr != nil {
        return emptyResponse, paramErr
    }

    paramErr = provider.SetDurationRange(msg.DurationMinimum, msg.DurationMaximum )
    if paramErr != nil {
        return emptyResponse, paramErr
    }

    paramErr = provider.SetProviderCancellationPenalty(msg.ProviderCancellationPenalty)
    if paramErr != nil {
        return emptyResponse, paramErr
    }

    paramErr = provider.SetConsumerCancellationPenalty(msg.ConsumerCancellationPenalty)
    if paramErr != nil {
        return emptyResponse, paramErr
    }

    paramErr = provider.SetAccessPolicy(msg.AccessPolicy)
    if paramErr != nil {
        return emptyResponse, paramErr
    }

    // Provider Grid values are OK to leave uninitialized
        // Unset Load is zero
        // Unset CheckpointBlock is zero

    newProvider := cc.NewProvider(provider)

    if _, playerOwned := owner.(*PlayerCache); playerOwned {
        providerPermissionId := GetObjectPermissionIDBytes(newProvider.ID(), owner.ID())
        cc.SetPermissions(providerPermissionId, types.PermProviderAll)
    }

	cc.CommitAll()
	return &types.MsgProviderResponse{}, nil
}
