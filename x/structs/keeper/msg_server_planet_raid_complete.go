package keeper

import (
	"context"
	"strconv"

	//"fmt"

	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
   message MsgPlanetRaidComplete {
     option (cosmos.msg.v1.signer) = "creator";

     string creator      = 1;
     string planetId     = 2;
     string playerId     = 3;
     string proof        = 4;
     string nonce        = 5;
   }

   message MsgPlanetRaidCompleteResponse { Planet planet = 1 [(gogoproto.nullable) = false]; }

*/

func (k msgServer) PlanetRaidComplete(goCtx context.Context, msg *types.MsgPlanetRaidComplete) (*types.MsgPlanetRaidCompleteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// Load Fleet
	fleet, fleetLoadError := cc.GetFleetById(msg.FleetId)
	if fleetLoadError != nil {
		return &types.MsgPlanetRaidCompleteResponse{}, fleetLoadError
	}

	// Check calling address can use Fleet
	/*
	   permissionError := fleet.GetOwner().CanBeHashedBy(msg.Creator)
	   if (permissionError != nil) {
	       return &types.MsgPlanetRaidCompleteResponse{}, permissionError
	   }
	*/

	// check that the fleet is Away
	if fleet.IsOnStation() {
		return &types.MsgPlanetRaidCompleteResponse{}, types.NewFleetStateError(fleet.GetFleetId(), "on_station", "raid_complete")
	}

	// check that forward pointer for the fleet is ""
	if fleet.GetFleet().LocationListForward != "" {
		return &types.MsgPlanetRaidCompleteResponse{}, types.NewFleetStateError(fleet.GetFleetId(), "not_first_in_queue", "raid_complete").WithPosition(0)
	}

	// check that the player is online
	if fleet.GetOwner().IsOffline() {
		return &types.MsgPlanetRaidCompleteResponse{}, types.NewPlayerPowerError(fleet.GetOwnerId(), "offline")
	}

	raidedPlanet := fleet.GetPlanet().GetPlanetId()
	blockStartRaidString := strconv.FormatUint(fleet.GetPlanet().GetBlockStartRaid(), 10)
	hashInput := msg.FleetId + "@" + fleet.GetPlanet().GetPlanetId() + "RAID" + blockStartRaidString + "NONCE" + msg.Nonce

	currentAge := uint64(ctx.BlockHeight()) - fleet.GetPlanet().GetBlockStartRaid()
	valid, achievedDifficulty := types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, fleet.GetPlanet().GetPlanetaryShield());
	if !valid {
		//_ = ctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: fleet.GetFleetId(), PlanetId: raidedPlanet, Status: types.RaidStatus_ongoing}})
		return &types.MsgPlanetRaidCompleteResponse{}, types.NewWorkFailureError("raid", fleet.GetFleetId(), hashInput).WithPlanet(fleet.GetPlanet().GetPlanetId())
	}

	// Award the Ore from the defender to attacker
	amountStolen := fleet.GetPlanet().GetOwner().GetStoredOre()
	fleet.GetOwner().StoredOreIncrement(amountStolen)
	fleet.GetPlanet().GetOwner().StoredOreEmpty()

	_ = ctx.EventManager().EmitTypedEvent(&types.EventOreTheft{&types.EventOreTheftDetail{VictimPlayerId: fleet.GetPlanet().GetOwnerId(), VictimPrimaryAddress: fleet.GetPlanet().GetOwner().GetPrimaryAddress(), ThiefPlayerId: fleet.GetOwnerId(), ThiefPrimaryAddress: fleet.GetOwner().GetPrimaryAddress(), Amount: amountStolen}})

	// Move the Fleet back to Station
	fleet.SetLocationToPlanet(fleet.GetOwner().GetPlanet())

	_ = ctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: fleet.GetFleetId(), PlanetId: raidedPlanet, Status: types.RaidStatus_raidSuccessful}})
    _ = ctx.EventManager().EmitTypedEvent(&types.EventHashSuccess{&types.EventHashSuccessDetail{CallerAddress: msg.Creator, Category: "raid", Difficulty: achievedDifficulty, ObjectId: msg.FleetId, PlanetId: raidedPlanet }})

	cc.CommitAll()
	return &types.MsgPlanetRaidCompleteResponse{Fleet: fleet.GetFleet(), Planet: fleet.GetPlanet().GetPlanet(), OreStolen: amountStolen}, nil
}
