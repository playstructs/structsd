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

    destination := k.GetPlanetCacheFromId(ctx, msg.DestinationLocationId)
    if (!destination.LoadPlanet()) {
        return &types.MsgFleetMoveResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Player (%s) is offline due to power", destination.GetOwnerId())
    }

    // Is the Fleet able to move?
    readinessError := fleet.PlanetMoveReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, fleet.GetOwnerId())
        return &types.MsgFleetMoveResponse{}, readinessError
    }

    fleet.SetLocationToPlanet(&destination)

    fleet.Commit()

	return &types.MsgFleetMoveResponse{Fleet: &fleet.Fleet}, nil
}
