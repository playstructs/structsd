package keeper

import (
	"context"
    "strconv"
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

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build actions requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, true)

    if (!player.IsOnline()){
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrSubstationOffline, "The player (%s) is offline ",player.Id)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionPlay))) {
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%s) not found", msg.StructId)
    }


    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo build structs for others, no permission to complete (%s)", structure.Id)
    }


    /* More garbage clown code rushed to make the testnet more interesting */
    // Check the Proof
    buildStartBlockString           := strconv.FormatUint(structure.BuildStartBlock , 10)
    hashInput                       := structure.Id + "BUILD" + buildStartBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.BuildStartBlock


    if (!types.HashBuildAndCheckBuildDifficulty(hashInput, msg.Proof, currentAge)) {
       return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructBuildComplete, "Work failure for input (%s) when trying to build Struct %s", hashInput, structure.Id)
    }

    // Try to bring online if there is room in the energy cap
    if (!player.CanSupportNewLoad(structure.PassiveDraw)) {
        return &types.MsgStructBuildCompleteResponse{}, sdkerrors.Wrapf(types.ErrStructBuildComplete, "Could not bring Struct %s online, player %s does not have enough power",structure.Id, player.Id)
    }

    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), structure.PassiveDraw)


    structure.SetStatus("ACTIVE")
    k.SetStruct(ctx, structure)


	return &types.MsgStructBuildCompleteResponse{Struct: structure}, nil
}
