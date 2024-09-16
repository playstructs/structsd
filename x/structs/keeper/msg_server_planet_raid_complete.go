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

    // Check calling address can use Fleet

    // check that the fleet is Away

    // check that forward pointer for the fleet is ""

    // check that the player is online


    blockStartRaidString    := strconv.FormatUint(fleet.GetPlanet().GetBlockStartRaid() , 10)
    hashInput               := msg.FleetId + '@' + fleet.GetPlanet().GetPlanetId() + "RAID" + blockStartRaidString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - fleet.GetPlanet().GetBlockStartRaid()
    if (!types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, fleet.GetPlanet().GetBlockStartRaid())) {
       return &types.MsgStructOreMinerStatusResponse{}, sdkerrors.Wrapf(types.ErrStructMine, "Work failure for input (%s) when trying to mine on Struct %s", hashInput, structure.GetStructId())
    }

    // Award the Ore from the defender to attacker

    // Move the Fleet back to Station


	return &types.MsgPlanetRaidCompleteResponse{Planet: planet.GetPlanet()}, nil
}
