package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructRefine(goCtx context.Context, msg *types.MsgStructRefine) (*types.MsgStructRefineResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct refine requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)

    if (!k.SubstationIsOnline(ctx, player.SubstationId)){
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrSubstationOffline, "The players substation (%d) is offline ",player.SubstationId)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionPlay))) {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.StructId)
    }

    if (structure.Type != "Refinery") {
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrStructRefine, "This struct (%d) has no refining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrStructRefine, "This struct (%d) is not online", msg.StructId)
    }


    if (structure.RefiningSystemStatus != "ACTIVE") {
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrStructRefine, "This Refining System for struct (%d) is inactive", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%d)", structure.Owner)
    }

    planet, planetFound := k.GetPlanet(ctx, structure.PlanetId)
    if (!planetFound) {
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrPlanetNotFound, "Planet (%d) was not found, which is actually a pretty big problem. Please tell an adult", structure.PlanetId)
    }

    if (planet.Status != 0) {
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrStructRefine, "Planet (%d) is already complete. Move on bud, no work to be done here", structure.PlanetId)
    }

    if (k.GetPlanetOreCount(ctx, planet.Id) == 0) {
        return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrStructRefine, "Planet (%d) is empty, nothing to refine", structure.PlanetId)
    }

    structIdString := strconv.FormatUint(structure.Id, 10)
    activeRefiningSystemBlockString := strconv.FormatUint(structure.ActiveRefiningSystemBlock , 10)
    hashInput := structIdString + "REFINE" + activeRefiningSystemBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.ActiveRefiningSystemBlock
    if (!types.HashBuildAndCheckActionDifficulty(hashInput, msg.Proof, currentAge)) {
       return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrStructRefine, "Work failure for input (%s) when trying to refine on Struct %d", hashInput, structure.Id)
    }

    // decrement the balance of ore for the planet
    k.DecreasePlanetOreCount(ctx, planet.Id)


    // Can't seem to add to a player account directly anymore with AddCoins
    // So instead we Mint the new alpha to the module and transfer into the account

    // Mint the new Alpha to the module
    newAlpha, _ := sdk.ParseCoinsNormalized("1alpha")
    k.bankKeeper.MintCoins(ctx, types.ModuleName, newAlpha)
    // Transfer the refined Alpha to the player
    playerAcc, _ := sdk.AccAddressFromBech32(player.PrimaryAddress)
    k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, newAlpha)

    structure.SetRefiningSystemActivationBlock(uint64(ctx.BlockHeight()))
    k.SetStruct(ctx, structure)

	return &types.MsgStructRefineResponse{Struct: structure}, nil
}
