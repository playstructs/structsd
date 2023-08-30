package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructMineInitiate(goCtx context.Context, msg *types.MsgStructMineInitiate) (*types.MsgStructMineInitiateResponse, error) {
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

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%s) not found", structIdString)
    }

    if (!structure.StructType != "Mining Rig") {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructMineInitiate, "This struct (%s) has no mining systems", structIdString)
    }

    if (!structure.Status != "ACTIVE") {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructMineInitiate, "This struct (%s) is not online", structIdString)
    }


    if (!structure.MiningSystemStatus != "INACTIVE") {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructMineInitiate, "This Mining System for struct (%s) is already active", structIdString)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       structIdString := strconv.FormatUint(structure.Owner, 10)
       return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%s)", structIdString)
    }


    // Check to see if the planet has Ore remaining
    planet, planetFound := k.GetPlanet(ctx, structure.PlanetId)
    if (!planetFound) {
        planetIdString := strconv.FormatUint(structure.PlanetId, 10)
        return &types.MsgStructBuildInitiateResponse{}, sdkerrors.Wrapf(types.ErrPlanetNotFound, "Planet (%s) was not found. Which is concerning in this case...", planetIdString)
    }

    _, err := k.PlayerIncrementLoad(ctx, player, structure.ActiveMiningSystemDraw)

    if (err != nil) {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructMineInitiate, "Not enough power to bring Mining System online for Struct (%s)", structIdString)
    }

    structure.SetMiningSystemStatus("ACTIVE")
    // TODO IDK FINISH THIS SOMEHOW.
    // I DON'T Think the actual proto messages exist yet or the CLI

    k.SetStruct(ctx, structure)

	return &types.MsgStructBuildInitiateResponse{Struct: structure}, nil
}
