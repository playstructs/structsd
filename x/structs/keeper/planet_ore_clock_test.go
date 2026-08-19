package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestOreMiningActivateDeactivate(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1000)

	planet := testAppendPlanet(k, ctx, types.Planet{Creator: "addr1", Owner: "player1"})
	mineClockId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreMine, planet.Id)
	mineQtyId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreMiningActiveQuantity, planet.Id)

	cc := k.NewCurrentContext(sdkCtx)
	cc.GetPlanet(planet.Id).OreMiningActivate()
	cc.CommitAll()
	require.Equal(t, uint64(1), k.GetPlanetAttribute(sdkCtx, mineQtyId))
	require.Equal(t, uint64(1000), k.GetPlanetAttribute(sdkCtx, mineClockId))

	cc2 := k.NewCurrentContext(sdkCtx.WithBlockHeight(1500))
	cc2.GetPlanet(planet.Id).OreMiningActivate()
	cc2.CommitAll()
	require.Equal(t, uint64(2), k.GetPlanetAttribute(sdkCtx, mineQtyId))
	require.Equal(t, uint64(1000), k.GetPlanetAttribute(sdkCtx, mineClockId), "second activate must not re-anchor")

	cc3 := k.NewCurrentContext(sdkCtx.WithBlockHeight(1600))
	cc3.GetPlanet(planet.Id).OreMiningDeactivate()
	cc3.CommitAll()
	require.Equal(t, uint64(1), k.GetPlanetAttribute(sdkCtx, mineQtyId))
	require.Equal(t, uint64(1000), k.GetPlanetAttribute(sdkCtx, mineClockId), "clock stays while counter > 0")

	cc4 := k.NewCurrentContext(sdkCtx.WithBlockHeight(1700))
	cc4.GetPlanet(planet.Id).OreMiningDeactivate()
	cc4.CommitAll()
	require.Equal(t, uint64(0), k.GetPlanetAttribute(sdkCtx, mineQtyId))
	require.Equal(t, uint64(1000), k.GetPlanetAttribute(sdkCtx, mineClockId), "clock is left alone; counter is the on/off signal")

	cc5 := k.NewCurrentContext(sdkCtx.WithBlockHeight(2000))
	cc5.GetPlanet(planet.Id).OreMiningActivate()
	cc5.CommitAll()
	require.Equal(t, uint64(1), k.GetPlanetAttribute(sdkCtx, mineQtyId))
	require.Equal(t, uint64(2000), k.GetPlanetAttribute(sdkCtx, mineClockId))
}

func TestOreRefiningActivateDeactivate(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(3000)

	planet := testAppendPlanet(k, ctx, types.Planet{Creator: "addr1", Owner: "player1"})
	refineClockId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreRefine, planet.Id)
	refineQtyId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreRefiningActiveQuantity, planet.Id)

	cc := k.NewCurrentContext(sdkCtx)
	cc.GetPlanet(planet.Id).OreRefiningActivate()
	cc.CommitAll()
	require.Equal(t, uint64(1), k.GetPlanetAttribute(sdkCtx, refineQtyId))
	require.Equal(t, uint64(3000), k.GetPlanetAttribute(sdkCtx, refineClockId))

	cc2 := k.NewCurrentContext(sdkCtx.WithBlockHeight(3500))
	cc2.GetPlanet(planet.Id).OreRefiningActivate()
	cc2.CommitAll()
	require.Equal(t, uint64(2), k.GetPlanetAttribute(sdkCtx, refineQtyId))
	require.Equal(t, uint64(3000), k.GetPlanetAttribute(sdkCtx, refineClockId))

	cc3 := k.NewCurrentContext(sdkCtx.WithBlockHeight(3600))
	cc3.GetPlanet(planet.Id).OreRefiningDeactivate()
	cc3.GetPlanet(planet.Id).OreRefiningDeactivate()
	cc3.CommitAll()
	require.Equal(t, uint64(0), k.GetPlanetAttribute(sdkCtx, refineQtyId))
	require.Equal(t, uint64(3000), k.GetPlanetAttribute(sdkCtx, refineClockId))
}

func TestPauseOreClocksForRaid(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	planet := testAppendPlanet(k, ctx, types.Planet{Creator: "addr1", Owner: "player1"})

	mineClockId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreMine, planet.Id)
	refineClockId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreRefine, planet.Id)
	mineQtyId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreMiningActiveQuantity, planet.Id)
	refineQtyId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreRefiningActiveQuantity, planet.Id)
	arrivedId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockRaiderArrived, planet.Id)

	t.Run("pre-raid clock keeps its age", func(t *testing.T) {
		// Clock at 100, raider arrived at 200, raid ends at 500.
		// pauseStart = max(100, 200) = 200; newClock = 100 + (500-200) = 400.
		k.SetPlanetAttribute(ctx, mineClockId, 100)
		k.SetPlanetAttribute(ctx, mineQtyId, 1)
		k.SetPlanetAttribute(ctx, arrivedId, 200)

		sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(500)
		cc := k.NewCurrentContext(sdkCtx)
		cc.GetPlanet(planet.Id).PauseOreClocksForRaid()
		cc.CommitAll()

		require.Equal(t, uint64(400), k.GetPlanetAttribute(sdkCtx, mineClockId))
	})

	t.Run("mid-raid clock lands at age zero", func(t *testing.T) {
		// Clock anchored mid-raid at 300, raider arrived at 200, raid ends at 500.
		// pauseStart = max(300, 200) = 300; newClock = 300 + (500-300) = 500.
		k.SetPlanetAttribute(ctx, refineClockId, 300)
		k.SetPlanetAttribute(ctx, refineQtyId, 1)
		k.SetPlanetAttribute(ctx, arrivedId, 200)

		sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(500)
		cc := k.NewCurrentContext(sdkCtx)
		cc.GetPlanet(planet.Id).PauseOreClocksForRaid()
		cc.CommitAll()

		require.Equal(t, uint64(500), k.GetPlanetAttribute(sdkCtx, refineClockId))
	})

	t.Run("zero counter skips shift", func(t *testing.T) {
		k.SetPlanetAttribute(ctx, mineClockId, 100)
		k.SetPlanetAttribute(ctx, mineQtyId, 0)
		k.SetPlanetAttribute(ctx, arrivedId, 200)

		sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(500)
		cc := k.NewCurrentContext(sdkCtx)
		cc.GetPlanet(planet.Id).PauseOreClocksForRaid()
		cc.CommitAll()

		require.Equal(t, uint64(100), k.GetPlanetAttribute(sdkCtx, mineClockId), "inactive clock must not shift")
	})

	t.Run("no raider arrived is a no-op", func(t *testing.T) {
		k.SetPlanetAttribute(ctx, mineClockId, 100)
		k.SetPlanetAttribute(ctx, mineQtyId, 1)
		k.ClearPlanetAttribute(ctx, arrivedId)

		sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(500)
		cc := k.NewCurrentContext(sdkCtx)
		cc.GetPlanet(planet.Id).PauseOreClocksForRaid()
		cc.CommitAll()

		require.Equal(t, uint64(100), k.GetPlanetAttribute(sdkCtx, mineClockId))
	})
}

func TestSetLocationListStartRaidOrePause(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1000)
	ctx = sdkCtx

	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        "cosmos1raidore",
		PrimaryAddress: "cosmos1raidore",
	})
	planet := testAppendPlanet(k, ctx, types.Planet{Creator: player.Creator, Owner: player.Id})
	player.PlanetId = planet.Id
	k.SetPlayer(ctx, player)

	mineClockId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreMine, planet.Id)
	mineQtyId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreMiningActiveQuantity, planet.Id)
	arrivedId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockRaiderArrived, planet.Id)
	k.SetPlanetAttribute(ctx, mineClockId, 800)
	k.SetPlanetAttribute(ctx, mineQtyId, 1)

	t.Run("first arrival anchors blockRaiderArrived", func(t *testing.T) {
		cc := k.NewCurrentContext(sdkCtx)
		cc.GetPlanet(planet.Id).SetLocationListStart("5-1")
		cc.CommitAll()

		got, _ := k.GetPlanet(sdkCtx, planet.Id)
		require.Equal(t, "5-1", got.LocationListStart)
		require.Equal(t, uint64(1000), k.GetPlanetAttribute(sdkCtx, arrivedId))
	})

	t.Run("promotion leaves blockRaiderArrived untouched", func(t *testing.T) {
		promoCtx := sdkCtx.WithBlockHeight(1200)
		cc := k.NewCurrentContext(promoCtx)
		planetState, _ := k.GetPlanet(promoCtx, planet.Id)
		require.Equal(t, "5-1", planetState.LocationListStart)

		cc.GetPlanet(planet.Id).SetLocationListStart("5-2")
		cc.CommitAll()

		got, _ := k.GetPlanet(promoCtx, planet.Id)
		require.Equal(t, "5-2", got.LocationListStart)
		require.Equal(t, uint64(1000), k.GetPlanetAttribute(promoCtx, arrivedId),
			"promotion must not reset blockRaiderArrived")
	})

	t.Run("raid end shifts clocks and clears marker", func(t *testing.T) {
		endCtx := sdkCtx.WithBlockHeight(1500)
		cc := k.NewCurrentContext(endCtx)
		cc.GetPlanet(planet.Id).SetLocationListStart("")
		cc.CommitAll()

		got, _ := k.GetPlanet(endCtx, planet.Id)
		require.Equal(t, "", got.LocationListStart)
		require.Equal(t, uint64(0), k.GetPlanetAttribute(endCtx, arrivedId))
		// pauseStart = max(800, 1000) = 1000; newClock = 800 + (1500-1000) = 1300
		require.Equal(t, uint64(1300), k.GetPlanetAttribute(endCtx, mineClockId))
	})
}

// TestOreCounterThroughStructLifecycle drives the counters through the real
// GoOnline/GoOffline path instead of the Activate/Deactivate helpers, covering
// the drift cases: a second rig coming online must not re-anchor the shared
// clock, and a rig that is already offline must not decrement a second time.
// That second guard is what keeps DestroyAndCommit from double-decrementing
// when it tears down an already-offline rig.
func TestOreCounterThroughStructLifecycle(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(500)

	player := types.Player{Creator: "cosmos1lifecycle", PrimaryAddress: "cosmos1lifecycle"}
	player = testAppendPlayer(k, sdkCtx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: player.Creator, Owner: player.Id})
	player.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, player)

	minerType := types.StructType{
		Id:              14,
		Type:            "Miner",
		Category:        types.ObjectType_planet,
		PlanetaryMining: types.TechPlanetaryMining_oreMiningRig,
	}
	k.SetStructType(sdkCtx, minerType)

	var minerIds []string
	for i := 0; i < 2; i++ {
		miner := testAppendStruct(k, sdkCtx, types.Struct{
			Creator: player.Creator, Owner: player.Id, Type: minerType.Id,
			LocationId: planet.Id, LocationType: types.ObjectType_planet,
		})
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, miner.Id)
		testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateBuilt))
		minerIds = append(minerIds, miner.Id)
	}

	mineClockId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreMine, planet.Id)
	mineQtyId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreMiningActiveQuantity, planet.Id)

	cc := k.NewCurrentContext(sdkCtx)
	cc.GetStruct(minerIds[0]).GoOnline()
	cc.CommitAll()
	require.Equal(t, uint64(1), k.GetPlanetAttribute(sdkCtx, mineQtyId))
	require.Equal(t, uint64(500), k.GetPlanetAttribute(sdkCtx, mineClockId), "first rig online anchors the clock")

	secondCtx := sdkCtx.WithBlockHeight(600)
	cc2 := k.NewCurrentContext(secondCtx)
	cc2.GetStruct(minerIds[1]).GoOnline()
	cc2.CommitAll()
	require.Equal(t, uint64(2), k.GetPlanetAttribute(secondCtx, mineQtyId))
	require.Equal(t, uint64(500), k.GetPlanetAttribute(secondCtx, mineClockId), "second rig must not discard accrued age")

	offlineCtx := sdkCtx.WithBlockHeight(700)
	cc3 := k.NewCurrentContext(offlineCtx)
	cc3.GetStruct(minerIds[1]).GoOffline()
	cc3.CommitAll()
	require.Equal(t, uint64(1), k.GetPlanetAttribute(offlineCtx, mineQtyId))

	repeatCtx := sdkCtx.WithBlockHeight(800)
	cc4 := k.NewCurrentContext(repeatCtx)
	cc4.GetStruct(minerIds[1]).GoOffline()
	cc4.CommitAll()
	require.Equal(t, uint64(1), k.GetPlanetAttribute(repeatCtx, mineQtyId), "offline rig must not decrement twice")

	lastCtx := sdkCtx.WithBlockHeight(900)
	cc5 := k.NewCurrentContext(lastCtx)
	cc5.GetStruct(minerIds[0]).GoOffline()
	cc5.CommitAll()
	require.Equal(t, uint64(0), k.GetPlanetAttribute(lastCtx, mineQtyId))
}

func TestOreMineRefineBlockedDuringRaid(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1_000_000_000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{
		Creator:        "cosmos1raidblock",
		PrimaryAddress: "cosmos1raidblock",
	}
	player = testAppendPlayer(k, sdkCtx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(100000))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{Creator: player.Creator, Owner: player.Id})
	player.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, player)

	planet.LocationListStart = "5-99"
	k.SetPlanet(sdkCtx, planet)

	oreAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, player.Id)
	k.SetGridAttribute(sdkCtx, oreAttrId, uint64(100))

	minerType := types.StructType{
		Id:                  14,
		Type:                "Miner",
		Category:            types.ObjectType_planet,
		PlanetaryMining:     types.TechPlanetaryMining_oreMiningRig,
		OreMiningDifficulty: 2,
	}
	k.SetStructType(sdkCtx, minerType)

	refineryType := types.StructType{
		Id:                    15,
		Type:                  "Refinery",
		Category:              types.ObjectType_planet,
		PlanetaryRefinery:     types.TechPlanetaryRefineries_oreRefinery,
		OreRefiningDifficulty: 2,
	}
	k.SetStructType(sdkCtx, refineryType)

	miner := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: player.Creator, Owner: player.Id, Type: minerType.Id,
		LocationId: planet.Id, LocationType: types.ObjectType_planet,
	})
	refinery := testAppendStruct(k, sdkCtx, types.Struct{
		Creator: player.Creator, Owner: player.Id, Type: refineryType.Id,
		LocationId: planet.Id, LocationType: types.ObjectType_planet,
	})

	for _, id := range []string{miner.Id, refinery.Id} {
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, id)
		testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateBuilt))
		testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateOnline))
	}

	k.SetPlanetAttribute(sdkCtx, keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreMine, planet.Id), 1)
	k.SetPlanetAttribute(sdkCtx, keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreMiningActiveQuantity, planet.Id), 1)
	k.SetPlanetAttribute(sdkCtx, keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreRefine, planet.Id), 1)
	k.SetPlanetAttribute(sdkCtx, keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreRefiningActiveQuantity, planet.Id), 1)

	_, err := ms.StructOreMinerComplete(wctx, &types.MsgStructOreMinerComplete{
		Creator: player.Creator, StructId: miner.Id, Nonce: "1", Proof: "deadbeef",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "under_raid")

	_, err = ms.StructOreRefineryComplete(wctx, &types.MsgStructOreRefineryComplete{
		Creator: player.Creator, StructId: refinery.Id, Nonce: "1", Proof: "deadbeef",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "under_raid")
}
