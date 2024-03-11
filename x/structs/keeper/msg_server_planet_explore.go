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

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Planet Exploration requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, false)


    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)

    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }


    if (!player.IsOnline()){
        return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",player.Id)
    }


    if (player.PlanetId != "") {
        // Check to see if the planet can be completed
        currentPlanet, currentPlanetFound := k.GetPlanet(ctx, player.PlanetId)
        if (!currentPlanetFound) {
            return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Planet (%s) was not found which in this case is extremely bad. Something horrible has happened", player.PlanetId)
        }

        if (!k.PlanetComplete(ctx, currentPlanet)) {
             return &types.MsgPlanetExploreResponse{}, sdkerrors.Wrapf(types.ErrPlanetExploration, "New Planet cannot be explored while current planet (%s) has Ore available for mining", player.PlanetId)
        }
    }

    planet := k.AppendPlanet(ctx, player)
    player.PlanetId = planet.Id

    k.SetPlayer(ctx, player)

	return &types.MsgPlanetExploreResponse{Planet: planet}, nil
}
