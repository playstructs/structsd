package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) FleetMove(goCtx context.Context, msg *types.MsgFleetMove) (*types.MsgFleetMoveResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


    // Load the fleet
    fleet, fleetLookupErr := k.GetFleetCacheFromId(ctx, msg.FleetId)
    if (fleetLookupErr != nil) {
        return &types.MsgFleetMoveResponse{}, fleetLookupErr
    }

    // Check address play permissions
    permissionError := fleet.GetOwner().CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgFleetMoveResponse{}, permissionError
    }

    if fleet.GetOwner().IsHalted() {
        return &types.MsgFleetMoveResponse{}, sdkerrors.Wrapf(types.ErrPlayerHalted, "Cannot perform actions while Player (%s) is Halted", fleet.GetOwnerId())
    }

    destination := k.GetPlanetCacheFromId(ctx, msg.DestinationLocationId)
    if (!destination.LoadPlanet()) {
        return &types.MsgFleetMoveResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Planet (%s) wasn't found", msg.DestinationLocationId)
    }

    // Is the Fleet able to move?
    readinessError := fleet.PlanetMoveReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, fleet.GetOwnerId())
        return &types.MsgFleetMoveResponse{}, readinessError
    }

    if fleet.GetLocationId() != msg.DestinationLocationId {
        if fleet.GetPlanet().GetLocationListStart() == msg.FleetId {
            _ = ctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: msg.FleetId, PlanetId: fleet.GetLocationId(), Status: types.RaidStatus_attackerRetreated}})
        }
    }

    fleet.SetLocationToPlanet(&destination)

    fleet.Commit()

	return &types.MsgFleetMoveResponse{Fleet: &fleet.Fleet}, nil
}
