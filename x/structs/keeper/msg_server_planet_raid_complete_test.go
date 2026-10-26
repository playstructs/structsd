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

/* TestMsgPlanetRaidCompleteRejectsHistoricalPlanet is the regression on a
 * completed planet still paying out its owner's live ore.
 *
 * Completing a planet leaves its Owner stamped and its record in the store:
 * PlanetExplore points the player at a new one and the old record stays
 * loadable, still naming them. The raid path never noticed, and every defence
 * that makes a live raid hard hangs off the owner's *current* planet:
 * RefreshRaidVulnerability is only ever called on owner.GetPlanet(), so a
 * historical planet's clock is started once by an arriving raider and never
 * reset again - it ages while the owner comes and goes from a planet they
 * actually occupy, decaying to a one-zero puzzle. IsDefenderCommandStructVulnerable
 * compounds it by asking whether the owner's fleet is on station, which for an
 * abandoned planet is a question about somewhere else entirely.
 *
 * Stored ore is a player attribute rather than a planet one, so the obsolete
 * planet paid out the victim's whole live balance.
 */
func TestMsgPlanetRaidCompleteRejectsHistoricalPlanet(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1_000_000_000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	attackerAcc := sdk.AccAddress("histraid_attacker_pad_addr_00000001")
	attacker := testAppendPlayer(k, sdkCtx, types.Player{
		Creator:        attackerAcc.String(),
		PrimaryAddress: attackerAcc.String(),
	})
	k.SetGridAttribute(sdkCtx,
		keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, attacker.Id), 100000)

	attackerHome := testAppendPlanet(k, sdkCtx, types.Planet{Creator: attacker.Creator, Owner: attacker.Id})
	attacker.PlanetId = attackerHome.Id
	k.SetPlayer(sdkCtx, attacker)

	victimAcc := sdk.AccAddress("histraid_victim_pad_addr_000000001")
	victim := testAppendPlayer(k, sdkCtx, types.Player{
		Creator:        victimAcc.String(),
		PrimaryAddress: victimAcc.String(),
	})
	k.SetGridAttribute(sdkCtx,
		keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, victim.Id), 100000)

	// The planet the victim used to live on: mined out, completed, and still
	// stamped with them as owner.
	abandoned := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator: victim.Creator,
		Owner:   victim.Id,
		Status:  types.PlanetStatus_complete,
	})

	// The planet they live on now. The victim's fleet sits here, away from the
	// abandoned one, which is what leaves that one permanently "vulnerable".
	current := testAppendPlanet(k, sdkCtx, types.Planet{Creator: victim.Creator, Owner: victim.Id})
	victim.PlanetId = current.Id
	victimFleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:        victim.Id,
		LocationId:   current.Id,
		LocationType: types.ObjectType_planet,
		Status:       types.FleetStatus_onStation,
	})
	victim.FleetId = victimFleet.Id
	k.SetPlayer(sdkCtx, victim)

	// The victim's live ore balance, which is a player attribute and therefore
	// has nothing to do with which planet is being raided.
	victimOreAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, victim.Id)
	k.SetGridAttribute(sdkCtx, victimOreAttrId, 900000)

	attackerOreAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, attacker.Id)

	attackerFleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:        attacker.Id,
		LocationId:   abandoned.Id,
		LocationType: types.ObjectType_planet,
		Status:       types.FleetStatus_away,
	})
	attacker.FleetId = attackerFleet.Id
	k.SetPlayer(sdkCtx, attacker)

	// A clock aged far enough that the puzzle has decayed to a single zero:
	// exactly what an unrefreshed historical planet hands an attacker.
	k.SetPlanetAttribute(sdkCtx,
		keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, abandoned.Id), 1)

	hashTemplate := fmt.Sprintf("%s@%sRAID1NONCE%%s", attackerFleet.Id, abandoned.Id)
	nonce, proof := testFindProof(hashTemplate, 1)

	_, err := ms.PlanetRaidComplete(wctx, &types.MsgPlanetRaidComplete{
		Creator: attacker.Creator,
		FleetId: attackerFleet.Id,
		Nonce:   nonce,
		Proof:   proof,
	})
	require.Error(t, err, "a planet its owner no longer lives on must not pay out their ore")
	require.Contains(t, err.Error(), "not_active")

	require.Equal(t, uint64(900000), k.GetGridAttribute(sdkCtx, victimOreAttrId),
		"the victim keeps every unit of their live balance")
	require.Equal(t, uint64(0), k.GetGridAttribute(sdkCtx, attackerOreAttrId),
		"and the attacker gains nothing")
}

// A planet nobody lives on any more must carry no raid state: neither one left
// behind by completion, nor one an arriving fleet tries to start afterwards.
func TestCompletedPlanetCarriesNoRaidClock(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(500)

	owner := testAppendPlayer(k, sdkCtx, types.Player{
		Creator:        sdk.AccAddress("raidclock_owner_pad1").String(),
		PrimaryAddress: sdk.AccAddress("raidclock_owner_pad1").String(),
	})

	planet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: owner.Creator, Owner: owner.Id})
	owner.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, owner)

	clockAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, planet.Id)
	arrivedAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockRaiderArrived, planet.Id)

	cc := k.NewCurrentContext(sdkCtx)
	planetCache := cc.GetPlanet(planet.Id)

	// Mine it out and complete it, with a clock left running.
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, planet.Id), 0)
	k.SetPlanetAttribute(sdkCtx, clockAttrId, 100)
	k.SetPlanetAttribute(sdkCtx, arrivedAttrId, 100)

	require.NoError(t, planetCache.AttemptComplete())
	cc.CommitAll()

	require.Zero(t, k.GetPlanetAttribute(sdkCtx, clockAttrId),
		"completing a planet must leave no raid clock behind")
	require.Zero(t, k.GetPlanetAttribute(sdkCtx, arrivedAttrId))

	// And a raider arriving afterwards must not start one.
	cc = k.NewCurrentContext(sdkCtx)
	cc.GetPlanet(planet.Id).SetLocationListStart("2-999")
	cc.CommitAll()

	require.Zero(t, k.GetPlanetAttribute(sdkCtx, clockAttrId),
		"an arrival on a completed planet must not start a clock that nothing will ever reset")
}
