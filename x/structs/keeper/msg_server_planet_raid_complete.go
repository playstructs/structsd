package keeper

import (
	"context"
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
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

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Load Fleet
    fleet, fleetLoadError := k.GetFleetCacheFromId(ctx, msg.FleetId)
    if (fleetLoadError != nil) {
        return &types.MsgPlanetRaidCompleteResponse{}, fleetLoadError
    }

    // Check calling address can use Fleet
    permissionError := fleet.GetOwner().CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgPlanetRaidCompleteResponse{}, permissionError
    }

    // check that the fleet is Away
    if fleet.IsOnStation() {
       return &types.MsgPlanetRaidCompleteResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Fleet cannot complete a Raid while On Station")
    }

    // check that forward pointer for the fleet is ""
    if (fleet.GetFleet().LocationListForward != "") {
        return &types.MsgPlanetRaidCompleteResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Fleet cannot complete a Raid unless it is the first in line")
    }

    // check that the player is online
    if fleet.GetOwner().IsOffline() {
        return &types.MsgPlanetRaidCompleteResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Fleet cannot complete a Raid unless the player is Online")
    }


    raidedPlanet            := fleet.GetPlanet().GetPlanetId()
    blockStartRaidString    := strconv.FormatUint(fleet.GetPlanet().GetBlockStartRaid() , 10)
    hashInput               := msg.FleetId + "@" + fleet.GetPlanet().GetPlanetId() + "RAID" + blockStartRaidString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - fleet.GetPlanet().GetBlockStartRaid()
    if (!types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, fleet.GetPlanet().GetPlanetaryShield())) {
        _ = ctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: fleet.GetFleetId(), PlanetId: raidedPlanet, Status: types.RaidStatus_ongoing}})
       return &types.MsgPlanetRaidCompleteResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Work failure for input (%s) when trying to complete a Raid on Planet %s", hashInput, fleet.GetPlanet().GetPlanetId())
    }

    // Award the Ore from the defender to attacker
    amountStolen := fleet.GetPlanet().GetOwner().GetStoredOre()
    fleet.GetOwner().StoredOreIncrement(amountStolen)
    fleet.GetPlanet().GetOwner().StoredOreEmpty()

    // Move the Fleet back to Station
    fleet.SetLocationToPlanet(fleet.GetOwner().GetPlanet())
    fleet.Commit()

    _ = ctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: fleet.GetFleetId(), PlanetId: raidedPlanet, Status: types.RaidStatus_raidSuccessful}})

	return &types.MsgPlanetRaidCompleteResponse{Fleet: fleet.GetFleet(), Planet: fleet.GetPlanet().GetPlanet(), OreStolen: amountStolen}, nil
}
