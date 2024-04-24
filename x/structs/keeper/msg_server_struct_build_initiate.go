package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructBuildInitiate(goCtx context.Context, msg *types.MsgStructBuildInitiate) (*types.MsgStructBuildInitiateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, true)

    if (!player.IsOnline()){
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The players substation (%s) is offline ",player.Id)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }


    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.PlanetId != msg.PlanetId) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo build structs for others, you should be building at %s", player.PlanetId)
    }

    planet, planetFound := k.GetPlanet(ctx, msg.PlanetId)
    if (!planetFound) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Planet (%s) was ot found. Building a Struct in a void might be tough", msg.PlanetId)
    }


    /* More garbage clown code rushed to make the testnet more interesting */
    if (msg.Slot >= planet.LandSlots) {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified doesn't have that slot available to build on", msg.PlanetId)
    }

    if (planet.Land[msg.Slot] != "") {
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The planet (%s) specified already has a struct on that slot", msg.PlanetId)
    }

    structure := k.AppendStruct(ctx, player, msg.StructType, planet, msg.Slot)
    planet.SetLandSlot(structure)
    k.SetPlanet(ctx, planet)

	return &types.MsgStructBuildInitiateResponse{Struct: structure}, nil
}
