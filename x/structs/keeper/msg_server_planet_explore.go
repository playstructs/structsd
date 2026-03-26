package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlanetExplore(goCtx context.Context, msg *types.MsgPlanetExplore) (*types.MsgPlanetExploreResponse, error) {
    emptyResponse := &types.MsgPlanetExploreResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    // Load the Player record
    player, playerLookupErr := cc.GetPlayer(msg.PlayerId)
    if (playerLookupErr != nil) {
        return emptyResponse, playerLookupErr
    }

    // Check address play permissions
    permissionError := player.CanBePlayedBy(callingPlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    // Is the Player online?
    readinessError := player.ReadinessCheck()
    if (readinessError != nil) {
        return emptyResponse, readinessError
    }

    // check if there is a planet currently
        // check that the planet can be completed
        // complete the previous planet
    if (player.HasPlanet()){
        planetCompletionError := player.GetPlanet().AttemptComplete()
        if (planetCompletionError != nil) {
            return emptyResponse, planetCompletionError
        }
    }

    planetExploreError := player.AttemptPlanetExplore()
    if (planetExploreError != nil) {
        return emptyResponse, planetExploreError
    }

    player.GetFleet().MigrateToNewPlanet(player.GetPlanet())

	cc.CommitAll()
	return &types.MsgPlanetExploreResponse{Planet: player.GetPlanet().GetPlanet()}, nil
}
