package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructBuildInitiate(goCtx context.Context, msg *types.MsgStructBuildInitiate) (*types.MsgStructBuildInitiateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)


    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionPlay) == 0) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }


    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.PlanetId != msg.PlanetId) {
        planetIdString := strconv.FormatUint(player.PlanetId, 10)
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo build structs for others, you should be building at %s", planetIdString)
    }

    planet, planetFound := k.GetPlanet(ctx, msg.PlanetId)
    if (!planetFound) {
        planetIdString := strconv.FormatUint(msg.PlanetId, 10)
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPlanetNotFound, "Planet (%s) was not found. Building a Struct in a void might be tough", planetIdString)
    }


    /* More garbage clown code rushed to make the testnet more interesting */
    if (msg.Slot >= planet.LandSlots) {
        planetIdString := strconv.FormatUint(msg.PlanetId, 10)
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet specified doesn't have that slot available to build on", planetIdString)
    }

    if (planet.Land[msg.Slot] > 0) {
        planetIdString := strconv.FormatUint(msg.PlanetId, 10)
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet specified already has a struct on that slot", planetIdString)
    }



    structure := k.AppendStruct(ctx, player, msg.StructType, planet, msg.Slot)
    planet.SetLandSlot(structure)
    k.SetPlanet(ctx, planet)


	return &types.MsgStructBuildInitiateResponse{Struct: structure}, nil
}
