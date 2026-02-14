package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlanetExplore(goCtx context.Context, msg *types.MsgPlanetExplore) (*types.MsgPlanetExploreResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Load the Player record
    player, playerLookupErr := cc.GetPlayer(msg.PlayerId)
    if (playerLookupErr != nil) {
        return &types.MsgPlanetExploreResponse{}, playerLookupErr
    }

    // Check address play permissions
    permissionError := player.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgPlanetExploreResponse{}, permissionError
    }

    // Is the Player online?
    readinessError := player.ReadinessCheck()
    if (readinessError != nil) {
        return &types.MsgPlanetExploreResponse{}, readinessError
    }

    // check if there is a planet currently
        // check that the planet can be completed
        // complete the previous planet
    if (player.HasPlanet()){
        planetCompletionError := player.GetPlanet().AttemptComplete()
        if (planetCompletionError != nil) {
            return &types.MsgPlanetExploreResponse{}, planetCompletionError
        }
    }

    planetExploreError := player.AttemptPlanetExplore()
    if (planetExploreError != nil) {
        return &types.MsgPlanetExploreResponse{}, planetExploreError
    }

    player.GetFleet().MigrateToNewPlanet(player.GetPlanet())

	cc.CommitAll()
	return &types.MsgPlanetExploreResponse{Planet: player.GetPlanet().GetPlanet()}, nil
}
