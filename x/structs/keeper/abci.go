package keeper

import (
	"context"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func (k *Keeper) BeginBlocker(ctx context.Context) {

    k.logger.Debug("Begin Block Processes")

    k.EmitEventTime(ctx)

    k.EventAllGenesis(ctx)

    k.StructSweepDestroyed(ctx)

    k.logger.Debug("Begin Block Complete")
}

// Called every block, update validator set
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	k.logger.Debug("End Block Processes")

	k.AgreementExpirations(ctx)

	/* Cascade all the possible failures across the grid
	 *
	 * This will mean that there will be some cases in which
	 * devices have one last block of power before shutting down
	 * but I think that's ok. We'll see how it goes in practice.
	 */
	k.GridCascade(ctx)

    k.logger.Debug("End Block Complete")

	return []abci.ValidatorUpdate{}, nil
}

func (k Keeper) EmitEventTime(ctx context.Context) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventTime{&types.EventTimeDetail{BlockHeight: ctxSDK.BlockHeight(), BlockTime: ctxSDK.HeaderInfo().Time.UTC() }})
}

func (k *Keeper) EventAllGenesis(ctx context.Context) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)

    if ctxSDK.BlockHeight() > 1 { return }

    k.logger.Info("Spewing Genesis Events into the Indexer")

	// Player
    players := k.GetAllPlayer(ctx)
    for _, player := range players {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})
    }

	// Address
    addresses := k.GetAllAddressExport(ctx)
    for _, address := range addresses {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address.Address, PlayerIndex: address.PlayerIndex, RegistrationStatus: types.RegistrationStatus_approved}})
	}

	// Permissions
    permissions := k.GetAllPermissionExport(ctx)
    for _, permission := range permissions {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPermission{&types.PermissionRecord{PermissionId: permission.PermissionId, Value: permission.Value}})
    }

	// Grid Attributes
	grids := k.GetAllGridExport(ctx)
    for _, grid := range grids {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGrid{&types.GridRecord{AttributeId: grid.AttributeId, Value: grid.Value}})
    }

	// Reactor
    reactors := k.GetAllReactor(ctx)
    for _, reactor := range reactors {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventReactor{Reactor: &reactor})
    }

	// Infusion
    infusions := k.GetAllInfusion(ctx)
    for _, infusion := range infusions {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventInfusion{Infusion: &infusion})
    }

	// Struct Type
    structTypes := k.GetAllStructType(ctx)
    for _, structType := range structTypes {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStructType{StructType: &structType})
    }

	// Banking


}