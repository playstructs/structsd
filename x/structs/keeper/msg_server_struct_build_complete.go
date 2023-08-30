package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructBuildComplete(goCtx context.Context, msg *types.MsgStructBuildComplete) (*types.MsgStructBuildCompleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build actions requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)


    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionPlay) == 0) {
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%s) not found", structIdString)
    }


    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       structIdString := strconv.FormatUint(structure.Owner, 10)
       return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo build structs for others, no permission to complete (%s)", structIdString)
    }


    /* More garbage clown code rushed to make the testnet more interesting */
    // Check the Proof
    structIdString                  := strconv.FormatUint(structure.Id, 10)
    buildStartBlockString           := strconv.FormatUint(structure.BuildStartBlock , 10)
    hashInput                       := structIdString + "BUILD" + buildStartBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.BuildStartBlock
    if (!types.HashBuildAndCheckBuildDifficulty(hashInput, msg.Proof, currentAge)) {
       return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Work failure for input (%s) when trying to build Struct %d", hashInput, structure.Id)
    }

    // Set the appropriate status
    if (msg.Activate) {
        _, err := k.PlayerIncrementLoad(ctx, player, structure.PassiveDraw)

        if (err == nil) {
            structure.SetStatus("ACTIVE")
        } else {
            structure.SetStatus("INACTIVE")
        }
    } else {
        structure.SetStatus("INACTIVE")
    }


    k.SetStruct(ctx, structure)


	return &types.MsgStructBuildCompleteResponse{Struct: structure}, nil
}
