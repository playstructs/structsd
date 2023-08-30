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

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)


    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionPlay) == 0) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.StructId)
    }

    if (structure.Type != "Mining Rig") {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "This struct (%d) has no mining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "This struct (%d) is not online", msg.StructId)
    }


    if (structure.MiningSystemStatus != "ACTIVE") {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "This Mining System for struct (%d) is inactive", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%d)", structure.Owner)
    }


    structIdString := strconv.FormatUint(structure.Owner, 10)
    activeMiningSystemBlockString := strconv.FormatUint(structure.ActiveMiningSystemBlock , 10)
    hashInput := structIdString + "MINE" + activeMiningSystemBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.ActiveMiningSystemBlock
    if (!types.HashBuildAndCheckActionDifficulty(hashInput, msg.Proof, currentAge)) {
       return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Work failure for input (%s) when trying to mine on Struct %d", hashInput, structure.Id)
    }

    // increment the balance of ore for the planet
    //



	return &types.MsgStructMineResponse{Struct: structure}, nil
}
