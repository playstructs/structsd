package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructRefine(goCtx context.Context, msg *types.MsgStructOreRefineryComplete) (*types.MsgStructOreRefineryStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (callingPlayerIndex == 0) {
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build actions requires Player account but none associated with %s", msg.Creator)
    }
    callingPlayerId := GetObjectID(types.ObjectType_player, callingPlayerIndex)

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structStatusAttributeId := GetStructAttributeIDByObjectId(types.StructAttributeType_status, msg.StructId)

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) not found", msg.StructId)
    }

    // Is the Struct online?
    if (k.StructAttributeFlagHasOneOf(ctx, structStatusAttributeId, uint64(types.StructStateOnline))) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) is offline. Activate it", msg.StructId)
    }

    if (callingPlayer.Id != structure.Owner) {
        // Check permissions on Creator on Planet
        playerPermissionId := GetObjectPermissionIDBytes(structure.Owner, callingPlayerId)
        if (!k.PermissionHasOneOf(ctx, playerPermissionId, types.PermissionPlay)) {
            return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling account (%s) has no play permissions on target player (%s)", callingPlayerId, structure.Owner)
        }
    }
    sudoPlayer, _ := k.GetPlayer(ctx, structure.Owner, true)
    if (!sudoPlayer.IsOnline()){
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",sudoPlayer.Id)
    }

    // Load Struct Type
    structType, structTypeFound := k.GetStructType(ctx, structure.Type)
    if (!structTypeFound) {
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", structure.Type)
    }

    // Check Sudo Player Charge
    // Maaaayybe we let the calling player use its charge but idk
    // Then people could have a stack of accounts to increase action throughput
    playerCharge := k.GetPlayerCharge(ctx, structure.Owner)
    if (playerCharge < structType.OreRefiningCharge) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for refining, but player (%s) only had %d", msg.StructTypeId, structType.ActivateCharge, structure.Owner, playerCharge)
    }

    if (structType.PlanetaryMining == types.TechPlanetaryMining_noPlanetaryMining) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) has no mining system", msg.StructId)
    }

    activeOreRefiningSystemBlock := k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structure.Id))
    // Is Struct Ore Miner running?
    if (ActiveOreRefiningSystemBlock == 0) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructOreRefineryStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) not refining", msg.StructId)
    }

    planet, planetFound := k.GetPlanet(ctx, structure.LocationId)
    if (!planetFound) {
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Planet (%s) was not found, which is actually a pretty big problem. Please tell an adult", structure.LocationId)
    }

    if (planet.Status == types.PlanetStatus_complete) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Planet (%s) is already complete. Move on bud, no work to be done here", structure.LocationId)
    }

    if (planet.PlayerOre == 0) {
        k.DischargePlayer(ctx, structure.Owner)
        return &types.MsgStructMineResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Planet (%s) has nothing to refine", structure.PlanetId)
    }

    activeOreRefiningSystemBlockString := strconv.FormatUint(activeOreRefiningSystemBlock , 10)
    hashInput := structure.Id + "REFINE" + activeOreRefiningSystemBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - activeOreRefiningSystemBlock
    if (!types.HashBuildAndCheckActionDifficulty(hashInput, msg.Proof, currentAge)) {
       return &types.MsgStructRefineResponse{}, sdkerrors.Wrapf(types.ErrStructRefine, "Work failure for input (%s) when trying to refine on Struct %s", hashInput, structure.Id)
    }

    // Got this far, let's reward the player with some fresh Alpha
    // Mint the new Alpha to the module
    newAlpha, _ := sdk.ParseCoinsNormalized("1alpha")
    k.bankKeeper.MintCoins(ctx, types.ModuleName, newAlpha)
    // Transfer the refined Alpha to the player
    playerAcc, _ := sdk.AccAddressFromBech32(player.PrimaryAddress)
    k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, playerAcc, newAlpha)
    // Remove the Ore
    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_ore, structure.Owner), 1)

    // Reset difficulty block
    k.SetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structure.Id), uint64(ctx.BlockHeight()))

    k.DischargePlayer(ctx, structure.Owner)

	return &types.MsgStructOreRefineryStatusResponse{Struct: structure}, nil
}
