package keeper

import (
	"context"
    "strconv"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) Sabotage(goCtx context.Context, msg *types.MsgSabotage) (*types.MsgSabotageResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgSabotageResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Sabotage requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, true)

    if (!player.IsOnline()){
        return &types.MsgSabotageResponse{}, sdkerrors.Wrapf(types.ErrSubstationOffline, "The player (%s) is offline ",player.Id)
    }


    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgSabotageResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%s) not found", msg.StructId)
    }


    switch structure.Type {
        case "Mining Rig":
            if (structure.MiningSystemStatus == "ACTIVE") {

                activeMiningSystemBlockString := strconv.FormatUint(structure.ActiveMiningSystemBlock , 10)
                hashInput := structure.Id + "SABOTAGE" + activeMiningSystemBlockString + "NONCE" + msg.Nonce

                currentAge := uint64(ctx.BlockHeight()) - structure.ActiveMiningSystemBlock
                if (!types.HashBuildAndCheckActionDifficultySabotage(hashInput, msg.Proof, currentAge, types.DifficultySabotageRangeMine)) {
                    return &types.MsgSabotageResponse{}, sdkerrors.Wrapf(types.ErrSabotage, "Work failure for input (%s) when trying to sabotage Struct %s", hashInput, structure.Id)
                }

                k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, structure.Owner), structure.ActiveMiningSystemDraw)
                structure.SetMiningSystemStatus("INACTIVE")
                structure.SetMiningSystemActivationBlock(0)
                k.SetStruct(ctx, structure)
            }
        case "Refinery":
            if (structure.RefiningSystemStatus == "ACTIVE") {

                activeRefiningSystemBlockString := strconv.FormatUint(structure.ActiveRefiningSystemBlock , 10)
                hashInput := structure.Id + "SABOTAGE" + activeRefiningSystemBlockString + "NONCE" + msg.Nonce

                currentAge := uint64(ctx.BlockHeight()) - structure.ActiveRefiningSystemBlock
                if (!types.HashBuildAndCheckActionDifficultySabotage(hashInput, msg.Proof, currentAge, types.DifficultySabotageRangeRefine)) {
                    return &types.MsgSabotageResponse{}, sdkerrors.Wrapf(types.ErrSabotage, "Work failure for input (%s) when trying to sabotage Struct %d", hashInput, structure.Id)
                }

                if (k.GetPlanetOreCount(ctx, structure.PlanetId) > 0) {
                    k.DecreasePlanetOreCount(ctx, structure.PlanetId)
                    k.IncreasePlanetOreCount(ctx, player.PlanetId)
                }
            }
        case "Small Generator":
            buildStartBlockString := strconv.FormatUint(structure.BuildStartBlock , 10)
            hashInput := structure.Id + "SABOTAGE" + buildStartBlockString + "NONCE" + msg.Nonce

            currentAge := uint64(ctx.BlockHeight()) - structure.BuildStartBlock
            if (!types.HashBuildAndCheckActionDifficultySabotage(hashInput, msg.Proof, currentAge, types.DifficultySabotageRangePower )) {
                return &types.MsgSabotageResponse{}, sdkerrors.Wrapf(types.ErrSabotage, "Work failure for input (%s) when trying to sabotage Struct %d", hashInput, structure.Id)
            }

            k.StructDestroy(ctx, structure)

    }



	return &types.MsgSabotageResponse{}, nil
}
