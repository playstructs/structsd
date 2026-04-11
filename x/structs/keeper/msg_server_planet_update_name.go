package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlanetUpdateName(goCtx context.Context, msg *types.MsgPlanetUpdateName) (*types.MsgPlanetUpdateResponse, error) {
	emptyResponse := &types.MsgPlanetUpdateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	k.AddressEmitActivity(ctx, msg.Creator)

	player, err := cc.GetPlayerByAddress(msg.Creator)
	if err != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "planet_update_name")
	}

	planet := cc.GetPlanet(msg.PlanetId)
	if !planet.PlanetLoaded {
		planet.LoadPlanet()
	}
	if !planet.PlanetLoaded {
		return emptyResponse, types.NewObjectNotFoundError("planet", msg.PlanetId)
	}

	permissionErr := planet.CanUpdateUGCBy(player)
	if permissionErr != nil {
		return emptyResponse, permissionErr
	}

	if err := types.ValidatePlanetName(msg.Name); err != nil {
		return emptyResponse, err
	}

	planet.SetName(msg.Name)

	cc.CommitAll()
	return &types.MsgPlanetUpdateResponse{}, nil
}
