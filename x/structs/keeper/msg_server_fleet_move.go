package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) FleetMove(goCtx context.Context, msg *types.MsgFleetMove) (*types.MsgFleetMoveResponse, error) {
    emptyResponse := &types.MsgFleetMoveResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    activePlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "fleet_move")
    }

    // Load the fleet
    fleet, fleetLookupErr := cc.GetFleetById(msg.FleetId)
    if (fleetLookupErr != nil) {
        return emptyResponse, fleetLookupErr
    }

    // Check address play permissions
    permissionError := fleet.GetOwner().CanBePlayedBy(activePlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    destination := cc.GetPlanet(msg.DestinationLocationId)
    if (!destination.LoadPlanet()) {
        return emptyResponse, types.NewObjectNotFoundError("planet", msg.DestinationLocationId)
    }

    // Is the Fleet able to move?
    readinessError := fleet.PlanetMoveReadinessCheck()
    if (readinessError != nil) {
        return emptyResponse, readinessError
    }

    if fleet.GetLocationId() != msg.DestinationLocationId {
        if fleet.GetPlanet().GetLocationListStart() == msg.FleetId {
            _ = ctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: msg.FleetId, PlanetId: fleet.GetLocationId(), Status: types.RaidStatus_attackerRetreated}})
        }
    }

    fleet.SetLocationToPlanet(destination)

	cc.CommitAll()
	return &types.MsgFleetMoveResponse{Fleet: &fleet.Fleet}, nil
}
