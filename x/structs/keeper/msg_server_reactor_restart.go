package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

/* ReactorRestart reconciles a reactor with live staking state when an unjailed
 * validator remains outside the active set and receives no bonded hook. It queues bounded reconciliation and is
 * permissionless because it can only restore state derived from Cosmos.
 */
func (k msgServer) ReactorRestart(goCtx context.Context, msg *types.MsgReactorRestart) (*types.MsgReactorRestartResponse, error) {
	emptyResponse := &types.MsgReactorRestartResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	valAddr, valErr := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if valErr != nil {
		return emptyResponse, types.NewReactorError("restart", "invalid_address").WithAddress(msg.ValidatorAddress, "validator")
	}

	reactorBytes, reactorFound := k.GetReactorBytesFromValidator(ctx, valAddr.Bytes())
	if !reactorFound {
		return emptyResponse, types.NewObjectNotFoundError("reactor", msg.ValidatorAddress)
	}

	reactor, found := k.GetReactorByBytes(ctx, reactorBytes)
	if !found {
		return emptyResponse, types.NewObjectNotFoundError("reactor", msg.ValidatorAddress)
	}
	k.EnqueueReactorRestart(ctx, reactor.Id)

	return &types.MsgReactorRestartResponse{}, nil
}
