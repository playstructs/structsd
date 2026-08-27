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

    activePlayer, err := cc.GetSigningPlayer(msg.Creator)
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

    /* A completed planet is not a place. It is mined out, its structs are
     * destroyed and its owner has moved on; nothing there can be raided,
     * attacked or built on. The record survives only because completion never
     * deletes it, and it keeps naming its former owner.
     *
     * Leaving it reachable is what let a fleet park on somebody's abandoned
     * planet and start a raid clock that nothing would ever reset - see
     * PlanetRaidComplete, which refuses an inactive planet, and
     * SetLocationListStart, which will not start a clock on one. This is the
     * outermost of the three: a fleet cannot get there at all.
     *
     * Only player-chosen destinations are gated. The internal moves -
     * PeaceDeal, a completed raid, MigrateToNewPlanet - send a fleet to its
     * owner's current planet, which is active by construction, and refusing
     * those would strand fleets rather than protect anyone.
     */
    if (!destination.IsActive()) {
        return emptyResponse, types.NewPlanetStateError(destination.GetPlanetId(), "not_active", "move")
    }

	if fleet.GetLocationId() == msg.DestinationLocationId {
		return &types.MsgFleetMoveResponse{Fleet: &fleet.Fleet}, nil
	}

    // Is the Fleet able to move?
    readinessError := fleet.PlanetMoveReadinessCheck()
    if (readinessError != nil) {
        return emptyResponse, readinessError
    }

    // Foreign destinations only: capacity is 1 + locationListExtra.
    if fleet.GetOwner().GetPlanetId() != destination.GetPlanetId() {
        if destination.GetLocationListCount() >= destination.GetLocationListCapacity() {
            return emptyResponse, types.NewFleetStateError(
                fleet.GetFleetId(), "queue_full", "move",
            ).WithPosition(destination.GetLocationListCount())
        }
    }

    // A moving fleet that heads its current planet's visitor queue is a raider
    // abandoning the raid. (The same-destination no-op returned earlier.)
    if fleet.GetPlanet().GetLocationListStart() == msg.FleetId {
        _ = ctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: msg.FleetId, PlanetId: fleet.GetLocationId(), Status: types.RaidStatus_attackerRetreated}})
    }

    fleet.SetLocationToPlanet(destination)

	cc.CommitAll()
	return &types.MsgFleetMoveResponse{Fleet: &fleet.Fleet}, nil
}
