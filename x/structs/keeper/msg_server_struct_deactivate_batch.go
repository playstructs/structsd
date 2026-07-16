package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) StructDeactivateBatch(goCtx context.Context, msg *types.MsgStructDeactivateBatch) (*types.MsgStructDeactivateBatchResponse, error) {
	emptyResponse := &types.MsgStructDeactivateBatchResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, err
	}

	batchSize := len(msg.StructId)
	if batchSize == 0 {
		return emptyResponse, types.NewParameterValidationError("structId", 0, "empty")
	}
	if batchSize > types.MaxStructDeactivateBatchSize {
		return emptyResponse, types.NewParameterValidationError("structId", uint64(batchSize), "above_maximum")
	}

	// Validate every struct before mutating any state. GoOffline has side
	// effects (raid-shield hook, allocation destruction, grid changes) that we
	// don't want to apply until the whole batch is confirmed eligible.
	seen := make(map[string]bool, batchSize)
	structures := make([]*StructCache, 0, batchSize)
	for _, structId := range msg.StructId {
		if seen[structId] {
			return emptyResponse, types.NewParameterValidationError("structId", 0, "duplicate")
		}
		seen[structId] = true

		structure := cc.GetStruct(structId)
		if eligibilityErr := structure.CanBeDeactivatedBy(callingPlayer); eligibilityErr != nil {
			return emptyResponse, eligibilityErr
		}

		structures = append(structures, structure)
	}

	deactivated := make([]types.Struct, 0, batchSize)
	for _, structure := range structures {
		structure.GoOffline()
		deactivated = append(deactivated, structure.GetStruct())
	}

	cc.CommitAll()
	return &types.MsgStructDeactivateBatchResponse{Structs: deactivated}, nil
}
