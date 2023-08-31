package keeper

import (
	"context"


	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructInfuse(goCtx context.Context, msg *types.MsgStructInfuse) (*types.MsgStructInfuseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct infuse requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)


    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionPlay) == 0) {
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.StructId)
    }

    if (structure.Type != "Small Generator") {
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "This struct (%d) has no power systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "This struct (%d) is not online", msg.StructId)
    }


    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%d)", structure.Owner)
    }

    planet, planetFound := k.GetPlanet(ctx, structure.PlanetId)
    if (!planetFound) {
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrPlanetNotFound, "Planet (%d) was not found, which is actually a pretty big problem. Please tell an adult", structure.PlanetId)
    }

    if (planet.Status != 0) {
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "Planet (%d) is already complete. Move on bud, no work to be done here", structure.PlanetId)
    }


    // Check the player primary address
    // Does it have any alpha?


    // Mint the new Alpha to the module
    infusionAmount, parseError := sdk.ParseCoinsNormalized(msg.InfuseAmount)
    if (parseError != nil ){
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "Infuse amount (%s) is invalid", msg.InfuseAmount)
    }

    if (infusionAmount[0].Denom != "alpha") {
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "Infuse amount (%s) is invalid, %s is not a fuel", msg.InfuseAmount, infusionAmount[0].Denom)
    }

    // Transfer the refined Alpha from the player
    playerAcc, _ := sdk.AccAddressFromBech32(player.PrimaryAddress)
    sendError := k.bankKeeper.SendCoinsFromAccountToModule(ctx, playerAcc, types.ModuleName, infusionAmount)

    if (sendError != nil){
        return &types.MsgStructInfuseResponse{}, sdkerrors.Wrapf(types.ErrStructInfuse, "Infuse failed", sendError )
    }
    k.bankKeeper.BurnCoins(ctx, types.ModuleName, infusionAmount)

    // need to set fuel PowerSystemFuel

    // Set energy powerSystemEnergy



	return &types.MsgStructInfuseResponse{}, nil
}
