package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) PlanetExplore(goCtx context.Context, msg *types.MsgPlanetExplore) (*types.MsgPlanetExploreResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Planet Exploration requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)


    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionPlay) == 0) {
        return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }


    if (player.PlanetId > 0) {
        // Check to see if the planet can be completed
        currentPlanet, currentPlanetFound := k.GetPlanet(ctx, player.PlanetId)
        if (!currentPlanetFound) {
            return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrPlanetNotFound, "Planet (%d) was not found which in this case is extremely bad. Something horrible has happened", player.PlanetId)
        }

        if (!k.PlanetComplete(ctx, currentPlanet)) {
             return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrPlanetExploration, "New Planet cannot be explored while current planet (%d) has Ore available for mining", player.PlanetId)
        }
    }

    planet := k.AppendPlanet(ctx, player)
    player.SetPlanetId(planet.Id)
    k.SetPlayer(ctx, player)

	return &types.MsgPlanetExploreResponse{Planet: planet}, nil
}
