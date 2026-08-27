package keeper

import (
	"context"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"


	"structs/x/structs/types"
)

/*
message MsgStructDefenseSet {
  option (cosmos.msg.v1.signer) = "creator";

  string creator              = 1;
  string defenderStructId     = 2;
  string protectedStructId    = 3;
}
*/

func (k msgServer) StructDefenseSet(goCtx context.Context, msg *types.MsgStructDefenseSet) (*types.MsgStructStatusResponse, error) {
    emptyResponse := &types.MsgStructStatusResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    // load struct
    structure := cc.GetStruct(msg.DefenderStructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(callingPlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    if !structure.LoadStruct(){
        return emptyResponse, types.NewObjectNotFoundError("struct", msg.DefenderStructId)
    }

    if structure.IsDestroyed() {
        return emptyResponse, types.NewStructStateError(msg.DefenderStructId, "destroyed", "active", "defense_set")
    }

    if structure.IsOffline() {
        return emptyResponse, types.NewStructStateError(msg.DefenderStructId, "offline", "online", "defense_set")
    }

    // Only struct types flagged as able to defend may register as a defender
    if defenseCapabilityError := structure.CanDefend(); defenseCapabilityError != nil {
        return emptyResponse, defenseCapabilityError
    }

    // Check Player Charge
    if (structure.GetOwner().GetCharge() < structure.GetStructType().DefendChangeCharge) {
        err := types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().DefendChangeCharge, structure.GetOwner().GetCharge(), "defend").WithStructType(structure.GetStructType().Id)
        return emptyResponse, err
    }

    if structure.GetOwner().IsOffline(){
        return emptyResponse, types.NewPlayerPowerError(structure.GetOwnerId(), "offline")
    }


    /* A struct cannot defend itself.
     *
     * IsProtecting below compares locations, and a struct is trivially
     * co-located with itself, so nothing else here would refuse it. The damage
     * is in the ordering: StructAttack resolves defender counters, then the
     * volley, then the target's own counter. A target registered as its own
     * defender is picked up in the first pass, so its counter lands before the
     * volley that provoked it - and if that counter destroys the attacker, the
     * volley is voided entirely and the target takes nothing. That inverts the
     * documented sequence, in which a target counters only after surviving the
     * shots.
     */
    if msg.DefenderStructId == msg.ProtectedStructId {
        return emptyResponse, types.NewStructLocationError(structure.GetStructType().Id, "", "self_defense").WithStruct(structure.GetStructId()).WithLocation("struct", msg.ProtectedStructId)
    }

    //load target
    protectedStructure := cc.GetStruct(msg.ProtectedStructId)
    if !protectedStructure.LoadStruct() {
        return emptyResponse, types.NewObjectNotFoundError("struct", msg.ProtectedStructId)
    }

    // Are they within defensive range? Single source of truth on StructCache so
    // the runtime defender filter in ResolveDefenders enforces the exact same
    // rule we register against here.
    if !structure.IsProtecting(protectedStructure) {
        return emptyResponse, types.NewStructLocationError(structure.GetStructType().Id, "", "not_in_range").WithStruct(structure.GetStructId()).WithLocation("struct", msg.ProtectedStructId)
    }

    k.SetStructDefender(ctx, msg.ProtectedStructId, protectedStructure.GetStruct().Index, structure.GetStructId())

    structure.GetOwner().Discharge()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{}, nil
}
