package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructMine(goCtx context.Context, msg *types.MsgStructMine) (*types.MsgStructMineResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct mining requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, true)

    if (!player.IsOnline()){
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",player.Id)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionPlay))) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%s) not found", msg.StructId)
    }

    if (structure.Type != "Mining Rig") {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "This struct (%s) has no mining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "This struct (%s) is not online", msg.StructId)
    }


    if (structure.MiningSystemStatus != "ACTIVE") {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "This Mining System for struct (%s) is inactive", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%s)", structure.Owner)
    }

    planet, planetFound := k.GetPlanet(ctx, structure.PlanetId)
    if (!planetFound) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrPlanetNotFound, "Planet (%s) was not found, which is actually a pretty big problem. Please tell an adult", structure.PlanetId)
    }

    if (planet.Status != 0) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Planet (%s) is already complete. Move on bud, no work to be done here", structure.PlanetId)
    }

    if (planet.OreRemaining == 0) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Planet (%s) is empty, nothing to mine", structure.PlanetId)
    }

    activeMiningSystemBlockString   := strconv.FormatUint(structure.ActiveMiningSystemBlock , 10)
    hashInput                       := structure.Id + "MINE" + activeMiningSystemBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.ActiveMiningSystemBlock
    if (!types.HashBuildAndCheckActionDifficulty(hashInput, msg.Proof, currentAge)) {
       return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Work failure for input (%s) when trying to mine on Struct %s", hashInput, structure.Id)
    }

    // Got this far, let's reward the player with some Ore
    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, structure.PlanetId), 1)
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, structure.Owner), 1)

    // Reset difficulty block
    structure.SetMiningSystemActivationBlock(uint64(ctx.BlockHeight()))
    k.SetStruct(ctx, structure)

	return &types.MsgStructMineResponse{Struct: structure}, nil
}
