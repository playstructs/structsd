package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestRaidClockRejectsFutureHeightBeforeOreTransfer(t *testing.T) {
	for _, tc := range []struct {
		name   string
		height int64
		start  uint64
		want   string
	}{
		{"future", 99, 100, "precedes start block"},
		{"zero_restart_height", 0, 100, "precedes start block"},
		{"maximum_start", 100, ^uint64(0), "precedes start block"},
		{"same_height", 100, 100, "work"},
		{"one_block_old", 101, 100, "work"},
		{"mature", 125, 100, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ms, wrapped := setupMsgServer(t)
			ctx := sdk.UnwrapSDKContext(wrapped).WithBlockHeight(tc.height)
			attacker := testAppendPlayer(k, ctx, types.Player{Creator: sdk.AccAddress("raid_clock_attacker_").String(), PrimaryAddress: sdk.AccAddress("raid_clock_attacker_").String()})
			victim := testAppendPlayer(k, ctx, types.Player{Creator: sdk.AccAddress("raid_clock_defender_").String(), PrimaryAddress: sdk.AccAddress("raid_clock_defender_").String()})
			home := testAppendPlanet(k, ctx, types.Planet{Owner: attacker.Id, Creator: attacker.Creator})
			target := testAppendPlanet(k, ctx, types.Planet{Owner: victim.Id, Creator: victim.Creator})
			fleet := testAppendFleet(k, ctx, types.Fleet{Owner: attacker.Id, LocationId: target.Id, LocationType: types.ObjectType_planet, Status: types.FleetStatus_away})
			attacker.PlanetId, attacker.FleetId = home.Id, fleet.Id
			victim.PlanetId = target.Id
			k.SetPlayer(ctx, attacker)
			k.SetPlayer(ctx, victim)
			target.LocationListStart, target.LocationListLast, target.LocationListCount = fleet.Id, fleet.Id, 1
			k.SetPlanet(ctx, target)
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attacker.Id), 100000)
			ore := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, victim.Id)
			attackerOre := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, attacker.Id)
			clock := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, target.Id)
			k.SetGridAttribute(ctx, ore, 50)
			k.SetPlanetAttribute(ctx, clock, tc.start)
			nonce, proof := testFindProof(fmt.Sprintf("%s@%sRAID%dNONCE%%s", fleet.Id, target.Id, tc.start), 1)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			_, err := ms.PlanetRaidComplete(ctx, &types.MsgPlanetRaidComplete{Creator: attacker.Creator, FleetId: fleet.Id, Nonce: nonce, Proof: proof})
			if tc.want == "" {
				require.NoError(t, err)
				require.Zero(t, k.GetGridAttribute(ctx, ore))
				require.Equal(t, uint64(50), k.GetGridAttribute(ctx, attackerOre))
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
			require.Equal(t, uint64(50), k.GetGridAttribute(ctx, ore))
			require.Zero(t, k.GetGridAttribute(ctx, attackerOre))
			require.Equal(t, tc.start, k.GetPlanetAttribute(ctx, clock))
			currentFleet, found := k.GetFleet(ctx, fleet.Id)
			require.True(t, found)
			require.Equal(t, fleet, currentFleet)
			currentPlanet, found := k.GetPlanet(ctx, target.Id)
			require.True(t, found)
			require.Equal(t, target, currentPlanet)
			for _, event := range ctx.EventManager().Events() {
				require.NotContains(t, []string{"structs.structs.EventOreTheft", "structs.structs.EventHashSuccess", "structs.structs.EventRaid"}, event.Type)
			}
		})
	}
}
