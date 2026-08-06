package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgPlanetRaidComplete(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1_000_000_000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	// Attacker player
	attackerAcc := sdk.AccAddress("attacker1234567890123456789012345678")
	attacker := types.Player{
		Creator:        attackerAcc.String(),
		PrimaryAddress: attackerAcc.String(),
	}
	attacker = testAppendPlayer(k, sdkCtx, attacker)
	attackerCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attacker.Id)
	k.SetGridAttribute(sdkCtx, attackerCapAttrId, uint64(100000))

	// Attacker's home planet
	homePlanet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: attacker.Creator, Owner: attacker.Id})
	attacker.PlanetId = homePlanet.Id
	k.SetPlayer(sdkCtx, attacker)

	// Target player with planet but no fleet: the defending Command Ship is
	// non-existent, so the planet is shieldsVulnerable.
	targetAcc := sdk.AccAddress("target12345678901234567890123456789")
	target := types.Player{
		Creator:        targetAcc.String(),
		PrimaryAddress: targetAcc.String(),
	}
	target = testAppendPlayer(k, sdkCtx, target)
	targetCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, target.Id)
	k.SetGridAttribute(sdkCtx, targetCapAttrId, uint64(100000))

	targetPlanet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: target.Creator, Owner: target.Id})
	target.PlanetId = targetPlanet.Id
	k.SetPlayer(sdkCtx, target)

	// Give target player some ore to steal
	targetOreAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, target.Id)
	k.SetGridAttribute(sdkCtx, targetOreAttrId, uint64(50))

	// Set BlockStartRaid on the target planet
	blockStartRaidAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, targetPlanet.Id)
	k.SetPlanetAttribute(sdkCtx, blockStartRaidAttrId, uint64(1))

	// Create attacker's fleet at the target planet
	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:        attacker.Id,
		LocationId:   targetPlanet.Id,
		LocationType: types.ObjectType_planet,
		Status:       types.FleetStatus_away,
	})
	attacker.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, attacker)

	t.Run("valid raid complete against non-existent command ship", func(t *testing.T) {
		hashTemplate := fmt.Sprintf("%s@%sRAID1NONCE%%s", fleet.Id, targetPlanet.Id)
		nonce, proof := testFindProof(hashTemplate, 1)

		resp, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: attacker.Creator,
			FleetId: fleet.Id,
			Nonce:   nonce,
			Proof:   proof,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// The vulnerability check must be a pure read: the defender must not
		// have a fleet (or pre-built Command Ship) conjured into existence.
		refreshedTarget, found := k.GetPlayer(sdkCtx, target.Id)
		require.True(t, found)
		require.Empty(t, refreshedTarget.FleetId, "raid complete must not auto-create a defender fleet")
		_, fleetFound := k.GetFleet(sdkCtx, fmt.Sprintf("%d-%d", types.ObjectType_fleet, target.Index))
		require.False(t, fleetFound, "raid complete must not write a defender fleet record")
	})

	t.Run("fleet not found", func(t *testing.T) {
		_, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: attacker.Creator,
			FleetId: "invalid-fleet",
			Nonce:   "test-nonce",
			Proof:   "test-proof",
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: "cosmos1noperms",
			FleetId: fleet.Id,
			Nonce:   "test-nonce",
			Proof:   "test-proof",
		})
		require.Error(t, err)
	})
}

// TestMsgPlanetRaidCompleteCommandShipGuards covers the v0.18.0
// shieldsVulnerable rules: the raid hashing puzzle can only be won while the
// defending Command Ship is offline, destroyed, or non-existent, and never
// while the raid vulnerability clock (blockStartRaid) is unset.
func TestMsgPlanetRaidCompleteCommandShipGuards(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1_000_000_000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	// Attacker player with home planet
	attackerAcc := sdk.AccAddress("attacker2234567890123456789012345678")
	attacker := types.Player{
		Creator:        attackerAcc.String(),
		PrimaryAddress: attackerAcc.String(),
	}
	attacker = testAppendPlayer(k, sdkCtx, attacker)
	attackerCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attacker.Id)
	k.SetGridAttribute(sdkCtx, attackerCapAttrId, uint64(100000))

	homePlanet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: attacker.Creator, Owner: attacker.Id})
	attacker.PlanetId = homePlanet.Id
	k.SetPlayer(sdkCtx, attacker)

	// Target player with planet, fleet, and a Command Ship
	targetAcc := sdk.AccAddress("target22345678901234567890123456789")
	target := types.Player{
		Creator:        targetAcc.String(),
		PrimaryAddress: targetAcc.String(),
	}
	target = testAppendPlayer(k, sdkCtx, target)
	targetCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, target.Id)
	k.SetGridAttribute(sdkCtx, targetCapAttrId, uint64(100000))

	targetPlanet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: target.Creator, Owner: target.Id})
	target.PlanetId = targetPlanet.Id
	k.SetPlayer(sdkCtx, target)

	commandShip := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: target.Creator,
		Owner:   target.Id,
		Type:    types.CommandStructTypeId,
	})
	commandShipStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, commandShip.Id)

	targetFleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:         target.Id,
		LocationId:    targetPlanet.Id,
		LocationType:  types.ObjectType_planet,
		Status:        types.FleetStatus_onStation,
		CommandStruct: commandShip.Id,
	})
	target.FleetId = targetFleet.Id
	k.SetPlayer(sdkCtx, target)

	targetOreAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, target.Id)

	blockStartRaidAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, targetPlanet.Id)

	// Attacker's fleet at the target planet
	attackerFleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:        attacker.Id,
		LocationId:   targetPlanet.Id,
		LocationType: types.ObjectType_planet,
		Status:       types.FleetStatus_away,
	})
	attacker.FleetId = attackerFleet.Id
	k.SetPlayer(sdkCtx, attacker)

	// resetAttackerFleet puts the attacker fleet back at the target planet
	// after a successful raid sends it home.
	resetAttackerFleet := func() {
		attackerFleet.LocationId = targetPlanet.Id
		attackerFleet.LocationType = types.ObjectType_planet
		attackerFleet.Status = types.FleetStatus_away
		attackerFleet.LocationListForward = ""
		attackerFleet.LocationListBackward = ""
		k.SetFleet(sdkCtx, attackerFleet)
	}

	setCommandShipStatus := func(status types.StructState) {
		k.SetStructAttribute(sdkCtx, commandShipStatusAttrId, uint64(status))
	}

	hashTemplate := fmt.Sprintf("%s@%sRAID1NONCE%%s", attackerFleet.Id, targetPlanet.Id)
	nonce, proof := testFindProof(hashTemplate, 1)

	t.Run("rejected while defender command ship online", func(t *testing.T) {
		setCommandShipStatus(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline)
		k.SetPlanetAttribute(sdkCtx, blockStartRaidAttrId, uint64(1))

		_, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: attacker.Creator,
			FleetId: attackerFleet.Id,
			Nonce:   nonce,
			Proof:   proof,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "shields_active")
	})

	t.Run("rejected when raid clock unset even with command ship destroyed", func(t *testing.T) {
		// This is the guard against the trivial-difficulty exploit: with
		// blockStartRaid == 0 the puzzle age would be the full chain height.
		setCommandShipStatus(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateDestroyed)
		k.SetPlanetAttribute(sdkCtx, blockStartRaidAttrId, uint64(0))

		zeroClockTemplate := fmt.Sprintf("%s@%sRAID0NONCE%%s", attackerFleet.Id, targetPlanet.Id)
		zeroNonce, zeroProof := testFindProof(zeroClockTemplate, 1)

		_, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: attacker.Creator,
			FleetId: attackerFleet.Id,
			Nonce:   zeroNonce,
			Proof:   zeroProof,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "raid_clock_unset")
	})

	t.Run("valid raid while defender command ship offline", func(t *testing.T) {
		setCommandShipStatus(types.StructStateMaterialized | types.StructStateBuilt)
		k.SetPlanetAttribute(sdkCtx, blockStartRaidAttrId, uint64(1))
		k.SetGridAttribute(sdkCtx, targetOreAttrId, uint64(50))

		resp, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: attacker.Creator,
			FleetId: attackerFleet.Id,
			Nonce:   nonce,
			Proof:   proof,
		})
		require.NoError(t, err)
		require.Equal(t, uint64(50), resp.OreStolen)
		require.Equal(t, uint64(0), k.GetGridAttribute(sdkCtx, targetOreAttrId))
	})

	t.Run("valid raid while defender command ship destroyed", func(t *testing.T) {
		resetAttackerFleet()
		setCommandShipStatus(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateDestroyed)
		k.SetPlanetAttribute(sdkCtx, blockStartRaidAttrId, uint64(1))
		k.SetGridAttribute(sdkCtx, targetOreAttrId, uint64(75))

		resp, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
			Creator: attacker.Creator,
			FleetId: attackerFleet.Id,
			Nonce:   nonce,
			Proof:   proof,
		})
		require.NoError(t, err)
		require.Equal(t, uint64(75), resp.OreStolen)
	})
}

// TestCommandShipRaidShieldHooks verifies that taking the defending Command
// Ship offline starts the raid vulnerability clock (blockStartRaid) and
// emits shieldsVulnerable, and that bringing it back online clears the
// clock again, while a raid is in progress on the owner's planet.
func TestCommandShipRaidShieldHooks(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)

	defenderAcc := sdk.AccAddress("defender1234567890123456789012345678")
	defender := types.Player{
		Creator:        defenderAcc.String(),
		PrimaryAddress: defenderAcc.String(),
	}
	defender = testAppendPlayer(k, sdkCtx, defender)
	defenderCapAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, defender.Id)
	k.SetGridAttribute(sdkCtx, defenderCapAttrId, uint64(100000))

	// Command Ship struct type (PassiveDraw small enough for capacity)
	k.SetStructType(sdkCtx, types.StructType{
		Id:             types.CommandStructTypeId,
		Type:           types.CommandStruct,
		Category:       types.ObjectType_fleet,
		PassiveDraw:    50,
		ActivateCharge: 1,
	})

	// A raid is in progress on the defender's planet: an enemy fleet sits
	// at the front of the raid queue.
	raiderFleetId := fmt.Sprintf("%d-%d", types.ObjectType_fleet, 999)
	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:           defender.Creator,
		Owner:             defender.Id,
		LocationListStart: raiderFleetId,
		LocationListLast:  raiderFleetId,
		LocationListCount: 1,
	})
	defender.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, defender)

	commandShip := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:      defender.Creator,
		Owner:        defender.Id,
		Type:         types.CommandStructTypeId,
		LocationType: types.ObjectType_fleet,
	})
	commandShipStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, commandShip.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, commandShipStatusAttrId, uint64(types.StructStateMaterialized))
	testSetStructAttributeFlagAdd(k, sdkCtx, commandShipStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, commandShipStatusAttrId, uint64(types.StructStateOnline))

	defenderFleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:         defender.Id,
		LocationId:    planet.Id,
		LocationType:  types.ObjectType_planet,
		Status:        types.FleetStatus_onStation,
		CommandStruct: commandShip.Id,
	})
	defender.FleetId = defenderFleet.Id
	k.SetPlayer(sdkCtx, defender)

	blockStartRaidAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, planet.Id)
	require.Equal(t, uint64(0), k.GetPlanetAttribute(sdkCtx, blockStartRaidAttrId), "clock starts unset while command ship is online")

	t.Run("command ship offline starts the vulnerability clock", func(t *testing.T) {
		deactivateCtx := sdkCtx.WithBlockHeight(1500)
		_, err := ms.StructDeactivate(deactivateCtx, &types.MsgStructDeactivate{
			Creator:  defender.Creator,
			StructId: commandShip.Id,
		})
		require.NoError(t, err)
		require.Equal(t, uint64(1500), k.GetPlanetAttribute(sdkCtx, blockStartRaidAttrId), "blockStartRaid anchors at the offline height")
	})

	t.Run("command ship back online clears the vulnerability clock", func(t *testing.T) {
		activateCtx := sdkCtx.WithBlockHeight(1600)
		_, err := ms.StructActivate(activateCtx, &types.MsgStructActivate{
			Creator:  defender.Creator,
			StructId: commandShip.Id,
		})
		require.NoError(t, err)
		require.Equal(t, uint64(0), k.GetPlanetAttribute(sdkCtx, blockStartRaidAttrId), "blockStartRaid clears when shields are restored")
	})
}
