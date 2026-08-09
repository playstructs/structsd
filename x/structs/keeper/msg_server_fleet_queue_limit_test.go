package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// setupRaidReadyPlayer creates a player with home planet, online capacity,
// charge, and an on-station fleet with an online Command Ship.
func setupRaidReadyPlayer(t *testing.T, k keeperlib.Keeper, ctx sdk.Context, seed string) (types.Player, types.Fleet, types.Planet) {
	t.Helper()

	acc := sdk.AccAddress([]byte(fmt.Sprintf("%-40s", seed)[:40]))
	player := types.Player{Creator: acc.String(), PrimaryAddress: acc.String()}
	player = testAppendPlayer(k, ctx, player)

	capAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capAttrId, uint64(100000))
	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(ctx, lastActionAttrId, uint64(ctx.BlockHeight())-100)

	home := testAppendPlanet(k, ctx, types.Planet{Creator: player.Creator, Owner: player.Id})
	player.PlanetId = home.Id
	k.SetPlayer(ctx, player)

	commandShip := testAppendStruct(k, ctx, types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         types.CommandStructTypeId,
		LocationType: types.ObjectType_fleet,
	})
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, commandShip.Id)
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateMaterialized))
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))

	fleet := testAppendFleet(k, ctx, types.Fleet{
		Owner:         player.Id,
		LocationId:    home.Id,
		LocationType:  types.ObjectType_planet,
		Status:        types.FleetStatus_onStation,
		CommandStruct: commandShip.Id,
	})
	player.FleetId = fleet.Id
	k.SetPlayer(ctx, player)

	return player, fleet, home
}

func TestFleetQueueLimit(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(5000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	k.SetStructType(sdkCtx, types.StructType{
		Id:       types.CommandStructTypeId,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	})

	defender, _, target := setupRaidReadyPlayer(t, k, sdkCtx, "queuelimit-defender")
	attacker1, fleet1, _ := setupRaidReadyPlayer(t, k, sdkCtx, "queuelimit-attacker1")
	attacker2, fleet2, _ := setupRaidReadyPlayer(t, k, sdkCtx, "queuelimit-attacker2")
	attacker3, fleet3, _ := setupRaidReadyPlayer(t, k, sdkCtx, "queuelimit-attacker3")
	_ = defender

	// Default extra=0 => capacity 1.
	_, err := ms.FleetMove(wctx, &types.MsgFleetMove{
		Creator:               attacker1.Creator,
		FleetId:               fleet1.Id,
		DestinationLocationId: target.Id,
	})
	require.NoError(t, err)

	gotTarget, _ := k.GetPlanet(sdkCtx, target.Id)
	require.Equal(t, uint64(1), gotTarget.LocationListCount)
	require.Equal(t, fleet1.Id, gotTarget.LocationListStart)
	require.Equal(t, fleet1.Id, gotTarget.LocationListLast)

	_, err = ms.FleetMove(wctx, &types.MsgFleetMove{
		Creator:               attacker2.Creator,
		FleetId:               fleet2.Id,
		DestinationLocationId: target.Id,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "queue_full")

	gotTarget, _ = k.GetPlanet(sdkCtx, target.Id)
	require.Equal(t, uint64(1), gotTarget.LocationListCount, "rejected move must not change count")

	// Head returns home: count drops and a new visitor can arrive.
	_, err = ms.FleetMove(wctx, &types.MsgFleetMove{
		Creator:               attacker1.Creator,
		FleetId:               fleet1.Id,
		DestinationLocationId: attacker1.PlanetId,
	})
	require.NoError(t, err)

	gotTarget, _ = k.GetPlanet(sdkCtx, target.Id)
	require.Equal(t, uint64(0), gotTarget.LocationListCount)
	require.Equal(t, "", gotTarget.LocationListStart)
	require.Equal(t, "", gotTarget.LocationListLast, "last cleared when sole visitor returns home")

	_, err = ms.FleetMove(wctx, &types.MsgFleetMove{
		Creator:               attacker2.Creator,
		FleetId:               fleet2.Id,
		DestinationLocationId: target.Id,
	})
	require.NoError(t, err)

	// Raise extra to 1 => capacity 2; a second visitor is allowed, a third is not.
	gotTarget, _ = k.GetPlanet(sdkCtx, target.Id)
	gotTarget.LocationListExtra = 1
	k.SetPlanet(sdkCtx, gotTarget)

	_, err = ms.FleetMove(wctx, &types.MsgFleetMove{
		Creator:               attacker1.Creator,
		FleetId:               fleet1.Id,
		DestinationLocationId: target.Id,
	})
	require.NoError(t, err)

	gotTarget, _ = k.GetPlanet(sdkCtx, target.Id)
	require.Equal(t, uint64(2), gotTarget.LocationListCount)

	_, err = ms.FleetMove(wctx, &types.MsgFleetMove{
		Creator:               attacker3.Creator,
		FleetId:               fleet3.Id,
		DestinationLocationId: target.Id,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "queue_full")
}

func TestFleetQueueLimitHomeMoveIgnoresCapacity(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(5100)
	wctx := sdk.WrapSDKContext(sdkCtx)

	k.SetStructType(sdkCtx, types.StructType{
		Id:       types.CommandStructTypeId,
		Type:     types.CommandStruct,
		Category: types.ObjectType_fleet,
	})

	player, fleet, home := setupRaidReadyPlayer(t, k, sdkCtx, "queuelimit-home")
	enemy, _, foreign := setupRaidReadyPlayer(t, k, sdkCtx, "queuelimit-foreign")
	_ = enemy

	_, err := ms.FleetMove(wctx, &types.MsgFleetMove{
		Creator:               player.Creator,
		FleetId:               fleet.Id,
		DestinationLocationId: foreign.Id,
	})
	require.NoError(t, err)

	foreignPlanet, _ := k.GetPlanet(sdkCtx, foreign.Id)
	require.Equal(t, uint64(1), foreignPlanet.LocationListCount)

	_, err = ms.FleetMove(wctx, &types.MsgFleetMove{
		Creator:               player.Creator,
		FleetId:               fleet.Id,
		DestinationLocationId: home.Id,
	})
	require.NoError(t, err)

	homePlanet, _ := k.GetPlanet(sdkCtx, home.Id)
	require.Equal(t, uint64(0), homePlanet.LocationListCount, "home fleets never occupy the raid queue")
	foreignPlanet, _ = k.GetPlanet(sdkCtx, foreign.Id)
	require.Equal(t, uint64(0), foreignPlanet.LocationListCount)
	require.Equal(t, "", foreignPlanet.LocationListStart, "start cleared when queue empties")
	require.Equal(t, "", foreignPlanet.LocationListLast, "last cleared when sole visitor leaves")
}
