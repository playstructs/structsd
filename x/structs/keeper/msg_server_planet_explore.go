package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) PlanetExplore(goCtx context.Context, msg *types.MsgPlanetExplore) (*types.MsgPlanetExploreResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Load the Player record
    player, playerLookupErr := k.GetPlayerCacheFromId(ctx, msg.PlayerId)
    if (playerLookupErr != nil) {
        return &types.MsgPlanetExploreResponse{}, playerLookupErr
    }

    // Check address play permissions
    permissionError := player.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgPlanetExploreResponse{}, permissionError
    }

    if player.IsHalted() {
        return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrPlayerHalted, "Cannot perform actions while Player (%s) is Halted", msg.PlayerId)
    }

    // Is the Player online?
    readinessError := player.ReadinessCheck()
    if (readinessError != nil) {
        k.DischargePlayer(ctx, player.GetPlayerId())
        return &types.MsgPlanetExploreResponse{}, readinessError
    }

    // check if there is a planet currently
        // check that the planet can be completed
        // complete the previous planet
    if (player.HasPlanet()){
        planetCompletionError := player.GetPlanet().AttemptComplete()
        if (planetCompletionError != nil) {
            k.DischargePlayer(ctx, player.GetPlayerId())
            return &types.MsgPlanetExploreResponse{}, planetCompletionError
        }
    }

    planetExploreError := player.AttemptPlanetExplore()
    if (planetExploreError != nil) {
        k.DischargePlayer(ctx, player.GetPlayerId())
        return &types.MsgPlanetExploreResponse{}, planetExploreError
    }

    player.GetFleet().ManualLoadOwner(&player)
    player.GetFleet().MigrateToNewPlanet(player.GetPlanet())

    player.Commit()

	return &types.MsgPlanetExploreResponse{Planet: player.GetPlanet().GetPlanet()}, nil
}
