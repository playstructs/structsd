package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) FleetMove(goCtx context.Context, msg *types.MsgFleetMove) (*types.MsgFleetMoveResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


    // Load the fleet
    fleet, fleetLookupErr := cc.GetFleetById(msg.FleetId)
    if (fleetLookupErr != nil) {
        return &types.MsgFleetMoveResponse{}, fleetLookupErr
    }

    // Check address play permissions
    permissionError := fleet.GetOwner().CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgFleetMoveResponse{}, permissionError
    }

    destination := cc.GetPlanet(msg.DestinationLocationId)
    if (!destination.LoadPlanet()) {
        return &types.MsgFleetMoveResponse{}, types.NewObjectNotFoundError("planet", msg.DestinationLocationId)
    }

    // Is the Fleet able to move?
    readinessError := fleet.PlanetMoveReadinessCheck()
    if (readinessError != nil) {
        return &types.MsgFleetMoveResponse{}, readinessError
    }

    if fleet.GetLocationId() != msg.DestinationLocationId {
        if fleet.GetPlanet().GetLocationListStart() == msg.FleetId {
            _ = ctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: msg.FleetId, PlanetId: fleet.GetLocationId(), Status: types.RaidStatus_attackerRetreated}})
        }
    }

    fleet.SetLocationToPlanet(destination)

	return &types.MsgFleetMoveResponse{Fleet: &fleet.Fleet}, nil
}
