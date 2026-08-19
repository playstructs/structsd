package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Destroying a struct does not delete it. DestroyAndCommit flags it Destroyed,
// leaves Built set and deliberately leaves it in its planet slot; only the
// BeginBlocker sweep, StructSweepDelay blocks later, clears the slot and removes
// the object. Everything in this file is about that window.
//
// Two things went wrong in it. Destruction replayed, because DestroyAndCommit had
// no idempotency guard and the slot it left populated kept the struct visible to
// AttemptComplete — so the owner's load and type count were released twice.
// And a destroyed struct still read as built and offline, so it passed
// ActivationReadinessCheck and came back online; the sweep then deleted it
// without taking it offline again, stranding the planetary shield and defensive
// counters GoOnline had re-added against a struct that no longer existed.

const (
	// Chosen so a second, wrongful decrement is visible rather than clamped away
	// at zero. Every fixture starts the owner at PlayerPassiveDraw, the load every
	// player carries, standing in for structs that must survive the destruction.
	testDestroyedBuildDraw   = 400
	testDestroyedPassiveDraw = 250
	testDestroyedShield      = 7
)

type destroyedStructFixture struct {
	t   *testing.T
	k   keeperlib.Keeper
	ms  types.MsgServer
	ctx sdk.Context

	player types.Player
	planet types.Planet
	fleet  types.Fleet

	// A planetary defense: it contributes to the planet's shield and its
	// defensive cannon count, which is what makes a phantom visible.
	defenseType types.StructType
}

func setupDestroyedStructFixture(t *testing.T) *destroyedStructFixture {
	t.Helper()

	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	playerAcc := sdk.AccAddress("destroyedguards_padding_address_1234")
	player := testAppendPlayer(k, ctx, types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	})

	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), 1_000_000)
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), types.PlayerPassiveDraw)
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id), 0)

	planet := testAppendPlanet(k, ctx, types.Planet{Creator: player.Creator, Owner: player.Id})

	fleet := testAppendFleet(k, ctx, types.Fleet{Owner: player.Id})
	fleet.LocationId = planet.Id
	fleet.LocationType = types.ObjectType_planet
	k.SetFleet(ctx, fleet)

	player.PlanetId = planet.Id
	player.FleetId = fleet.Id
	k.SetPlayer(ctx, player)

	defenseType := types.StructType{
		Id:        1,
		Type:      "Ore Bunker",
		Category:  types.ObjectType_planet,
		MaxHealth: 100,

		BuildDraw:   testDestroyedBuildDraw,
		PassiveDraw: testDestroyedPassiveDraw,

		OreReserveDefenses:          types.TechOreReserveDefenses_oreBunker,
		PlanetaryShieldContribution: testDestroyedShield,
		PlanetaryDefenses:           types.TechPlanetaryDefenses_defensiveCannon,
	}
	k.SetStructType(ctx, defenseType)

	return &destroyedStructFixture{
		t:           t,
		k:           k,
		ms:          ms,
		ctx:         ctx,
		player:      player,
		planet:      planet,
		fleet:       fleet,
		defenseType: defenseType,
	}
}

// appendStruct puts a struct in the planet's land slot and gives the owner the
// build reservation InitiateStruct would have taken, plus a type count of two so
// that a wrongful second decrement lands on a number that can still go down.
func (f *destroyedStructFixture) appendStruct(built bool) types.Struct {
	f.t.Helper()

	planet, found := f.k.GetPlanet(f.ctx, f.planet.Id)
	require.True(f.t, found)
	slot := uint64(len(planet.Land))

	structure := testAppendStruct(f.k, f.ctx, types.Struct{
		Creator:        f.player.Creator,
		Owner:          f.player.Id,
		Type:           f.defenseType.Id,
		LocationId:     f.planet.Id,
		LocationType:   types.ObjectType_planet,
		OperatingAmbit: types.Ambit_land,
		Slot:           slot,
	})

	planet.Land = append(planet.Land, structure.Id)
	f.k.SetPlanet(f.ctx, planet)

	status := uint64(types.StructStateMaterialized)
	if built {
		status |= uint64(types.StructStateBuilt)
	}
	f.k.SetStructAttribute(f.ctx, f.statusAttr(structure.Id), status)
	f.k.SetStructAttribute(f.ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, structure.Id), f.defenseType.MaxHealth)
	f.k.SetStructAttribute(f.ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structure.Id), 1)

	f.k.SetStructAttribute(f.ctx, f.typeCountAttr(), 2)

	// Build initiation reserves BuildDraw; completing the build releases it and
	// GoOnline takes PassiveDraw instead. A built struct in this fixture is
	// brought online by the caller, so only the unbuilt case carries a reservation.
	if !built {
		loadAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, f.player.Id)
		f.k.SetGridAttribute(f.ctx, loadAttrId, f.k.GetGridAttribute(f.ctx, loadAttrId)+testDestroyedBuildDraw)
	}

	return structure
}

func (f *destroyedStructFixture) statusAttr(structId string) string {
	return keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId)
}

func (f *destroyedStructFixture) typeCountAttr() string {
	return keeperlib.GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, f.player.Id, f.defenseType.Id)
}

func (f *destroyedStructFixture) structsLoad() uint64 {
	return f.k.GetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, f.player.Id))
}

func (f *destroyedStructFixture) typeCount() uint64 {
	return f.k.GetStructAttribute(f.ctx, f.typeCountAttr())
}

func (f *destroyedStructFixture) planetaryShield() uint64 {
	return f.k.GetPlanetAttribute(f.ctx, keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, f.planet.Id))
}

func (f *destroyedStructFixture) defensiveCannons() uint64 {
	return f.k.GetPlanetAttribute(f.ctx, keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_defensiveCannonQuantity, f.planet.Id))
}

func (f *destroyedStructFixture) isDestroyed(structId string) bool {
	return testStructAttributeFlagHasAll(f.k, f.ctx, f.statusAttr(structId), uint64(types.StructStateDestroyed))
}

func (f *destroyedStructFixture) isOnline(structId string) bool {
	return testStructAttributeFlagHasAll(f.k, f.ctx, f.statusAttr(structId), uint64(types.StructStateOnline))
}

// withCache runs fn against a live CurrentContext and commits, which is how the
// tests reach cache methods a handler would otherwise gate.
func (f *destroyedStructFixture) withCache(fn func(cc *keeperlib.CurrentContext)) {
	cc := f.k.NewCurrentContext(f.ctx)
	fn(cc)
	cc.CommitAll()
}

// emptyPlanetOre makes the planet completable, which is the precondition
// PlanetExplore checks before destroying everything on it.
func (f *destroyedStructFixture) emptyPlanetOre() {
	f.k.SetGridAttribute(f.ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, f.planet.Id), 0)
}

// TestStructDestroy_IsIdempotent is the narrowest statement of the fix: the
// second call must change nothing. Both quantities it touches saturate at zero
// instead of erroring, so a replay is silent, which is why it went unnoticed.
func TestStructDestroy_IsIdempotent(t *testing.T) {
	f := setupDestroyedStructFixture(t)
	structure := f.appendStruct(false)

	f.withCache(func(cc *keeperlib.CurrentContext) {
		cc.GetStruct(structure.Id).DestroyAndCommit()
	})

	loadAfterFirst := f.structsLoad()
	typeCountAfterFirst := f.typeCount()
	require.Equal(t, uint64(types.PlayerPassiveDraw), loadAfterFirst,
		"destruction must release exactly the build reservation")
	require.Equal(t, uint64(1), typeCountAfterFirst)
	require.True(t, f.isDestroyed(structure.Id))

	f.withCache(func(cc *keeperlib.CurrentContext) {
		cc.GetStruct(structure.Id).DestroyAndCommit()
	})

	require.Equal(t, loadAfterFirst, f.structsLoad(),
		"a second destruction released the build reservation again, freeing capacity the player never paid for")
	require.Equal(t, typeCountAfterFirst, f.typeCount(),
		"a second destruction dropped the type count again, hiding a struct that still exists")
}

// TestStructDestroyedGuards_CancelThenExploreReleasesLoadOnce is the reported
// exploit, end to end and through the message server. Cancelling a build leaves
// the struct in its planet slot, and completing the planet walks every occupied
// slot, so the same struct is destroyed twice in one block.
func TestStructDestroyedGuards_CancelThenExploreReleasesLoadOnce(t *testing.T) {
	f := setupDestroyedStructFixture(t)
	structure := f.appendStruct(false)

	_, err := f.ms.StructBuildCancel(f.ctx, &types.MsgStructBuildCancel{
		Creator:  f.player.Creator,
		StructId: structure.Id,
	})
	require.NoError(t, err)
	require.True(t, f.isDestroyed(structure.Id))

	loadAfterCancel := f.structsLoad()
	typeCountAfterCancel := f.typeCount()
	require.Equal(t, uint64(types.PlayerPassiveDraw), loadAfterCancel)
	require.Equal(t, uint64(1), typeCountAfterCancel)

	// The slot the cancelled struct still occupies is what PlanetExplore walks.
	planet, found := f.k.GetPlanet(f.ctx, f.planet.Id)
	require.True(t, found)
	require.Contains(t, planet.Land, structure.Id,
		"a cancelled build stays in its slot until the sweep; without that this test proves nothing")

	f.emptyPlanetOre()
	_, err = f.ms.PlanetExplore(f.ctx, &types.MsgPlanetExplore{
		Creator:  f.player.Creator,
		PlayerId: f.player.Id,
	})
	require.NoError(t, err)

	require.Equal(t, loadAfterCancel, f.structsLoad(),
		"cancel then explore released the same build reservation twice, eroding the load backing the player's surviving structs")
	require.Equal(t, typeCountAfterCancel, f.typeCount(),
		"cancel then explore dropped the same type count twice")
}

// TestStructDestroyedGuards_ActivateRejected covers the second leg. Destruction
// leaves Built set and only removes Online, so before the fix a destroyed struct
// satisfied every activation check.
func TestStructDestroyedGuards_ActivateRejected(t *testing.T) {
	f := setupDestroyedStructFixture(t)
	structure := f.appendStruct(true)

	f.withCache(func(cc *keeperlib.CurrentContext) {
		cc.GetStruct(structure.Id).GoOnline()
	})
	require.Equal(t, uint64(types.PlanetaryShieldBase+testDestroyedShield), f.planetaryShield())
	require.Equal(t, uint64(1), f.defensiveCannons())

	f.withCache(func(cc *keeperlib.CurrentContext) {
		cc.GetStruct(structure.Id).DestroyAndCommit()
	})

	// Destruction took it offline, so its contributions are already gone.
	require.Equal(t, uint64(types.PlanetaryShieldBase), f.planetaryShield())
	require.Equal(t, uint64(0), f.defensiveCannons())
	require.False(t, f.isOnline(structure.Id))

	loadBefore := f.structsLoad()

	_, err := f.ms.StructActivate(f.ctx, &types.MsgStructActivate{
		Creator:  f.player.Creator,
		StructId: structure.Id,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "destroyed")

	require.False(t, f.isOnline(structure.Id), "a destroyed struct must not come back online")
	require.Equal(t, uint64(types.PlanetaryShieldBase), f.planetaryShield(),
		"reactivation re-raised the planetary shield for a struct queued for deletion")
	require.Equal(t, uint64(0), f.defensiveCannons(),
		"reactivation restored a defensive cannon for a struct queued for deletion")
	require.Equal(t, loadBefore, f.structsLoad())
}

// TestStructDestroyedGuards_NoPhantomSurvivesTheSweep is the whole escalation:
// the struct is deleted for good and nothing it contributed outlives it.
func TestStructDestroyedGuards_NoPhantomSurvivesTheSweep(t *testing.T) {
	f := setupDestroyedStructFixture(t)
	structure := f.appendStruct(true)

	f.withCache(func(cc *keeperlib.CurrentContext) {
		cc.GetStruct(structure.Id).GoOnline()
	})
	f.withCache(func(cc *keeperlib.CurrentContext) {
		cc.GetStruct(structure.Id).DestroyAndCommit()
	})

	_, err := f.ms.StructActivate(f.ctx, &types.MsgStructActivate{
		Creator:  f.player.Creator,
		StructId: structure.Id,
	})
	require.Error(t, err)

	sweepCtx := f.ctx.WithBlockHeight(f.ctx.BlockHeight() + types.StructSweepDelay)
	f.k.StructSweepDestroyed(sweepCtx)

	_, found := f.k.GetStruct(f.ctx, structure.Id)
	require.False(t, found, "the sweep must remove the struct")

	require.Equal(t, uint64(types.PlanetaryShieldBase), f.planetaryShield(),
		"planetary shield outlived the struct that raised it")
	require.Equal(t, uint64(0), f.defensiveCannons(),
		"defensive cannon count outlived the struct that provided it")
	require.Equal(t, uint64(types.PlayerPassiveDraw), f.structsLoad(),
		"the deleted struct is still drawing against the owner's capacity")

	planet, found := f.k.GetPlanet(f.ctx, f.planet.Id)
	require.True(t, found)
	require.NotContains(t, planet.Land, structure.Id, "the sweep must free the slot")
}

// TestStructDestroyedGuards_SweepBackstopsAStillOnlineStruct exercises the last
// line of defense. The guards above are what stop a destroyed struct from coming
// back online, so this drives the cache directly to stand in for a path nobody
// has found yet, and asserts the sweep refuses to delete an online struct on top
// of its own contributions.
func TestStructDestroyedGuards_SweepBackstopsAStillOnlineStruct(t *testing.T) {
	f := setupDestroyedStructFixture(t)
	structure := f.appendStruct(true)

	f.withCache(func(cc *keeperlib.CurrentContext) {
		cc.GetStruct(structure.Id).DestroyAndCommit()
	})
	require.True(t, f.isDestroyed(structure.Id))

	// The bypass: reactivate the rubble without going through a handler.
	f.withCache(func(cc *keeperlib.CurrentContext) {
		cc.GetStruct(structure.Id).GoOnline()
	})
	require.True(t, f.isOnline(structure.Id))
	require.Equal(t, uint64(types.PlanetaryShieldBase+testDestroyedShield), f.planetaryShield(),
		"the bypass is supposed to create a phantom; if it did not, this test proves nothing")

	sweepCtx := f.ctx.WithBlockHeight(f.ctx.BlockHeight() + types.StructSweepDelay)
	f.k.StructSweepDestroyed(sweepCtx)

	_, found := f.k.GetStruct(f.ctx, structure.Id)
	require.False(t, found)
	require.Equal(t, uint64(types.PlanetaryShieldBase), f.planetaryShield(),
		"the sweep deleted an online struct without reversing its planetary shield contribution")
	require.Equal(t, uint64(0), f.defensiveCannons(),
		"the sweep deleted an online struct without reversing its defensive cannon")
}

// TestStructDestroyedGuards_HandlersRejectDestroyed walks the handlers that had
// no state check of their own. StructBuildComplete is the one the report missed:
// it rejected only IsBuilt, so a cancelled build could be completed inside the
// window, setting Built on an object already queued for deletion.
func TestStructDestroyedGuards_HandlersRejectDestroyed(t *testing.T) {
	t.Run("build complete", func(t *testing.T) {
		f := setupDestroyedStructFixture(t)
		structure := f.appendStruct(false)

		f.withCache(func(cc *keeperlib.CurrentContext) {
			cc.GetStruct(structure.Id).DestroyAndCommit()
		})

		_, err := f.ms.StructBuildComplete(f.ctx, &types.MsgStructBuildComplete{
			Creator:  f.player.Creator,
			StructId: structure.Id,
			Nonce:    "0",
			Proof:    "0",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "destroyed")
		require.False(t, testStructAttributeFlagHasAll(f.k, f.ctx, f.statusAttr(structure.Id), uint64(types.StructStateBuilt)),
			"a struct queued for deletion must not become built")
	})

	t.Run("build cancel", func(t *testing.T) {
		f := setupDestroyedStructFixture(t)
		structure := f.appendStruct(false)

		_, err := f.ms.StructBuildCancel(f.ctx, &types.MsgStructBuildCancel{
			Creator:  f.player.Creator,
			StructId: structure.Id,
		})
		require.NoError(t, err)

		_, err = f.ms.StructBuildCancel(f.ctx, &types.MsgStructBuildCancel{
			Creator:  f.player.Creator,
			StructId: structure.Id,
		})
		require.Error(t, err, "cancelling an already cancelled build reported success for work that did not happen")
		require.Contains(t, err.Error(), "destroyed")
	})

	t.Run("move", func(t *testing.T) {
		f := setupDestroyedStructFixture(t)
		structure := f.appendStruct(true)

		f.withCache(func(cc *keeperlib.CurrentContext) {
			cc.GetStruct(structure.Id).DestroyAndCommit()
		})

		_, err := f.ms.StructMove(f.ctx, &types.MsgStructMove{
			Creator:      f.player.Creator,
			StructId:     structure.Id,
			LocationType: types.ObjectType_planet,
			Ambit:        types.Ambit_air,
			Slot:         0,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "destroyed")
	})

	t.Run("defense set and clear", func(t *testing.T) {
		f := setupDestroyedStructFixture(t)
		defender := f.appendStruct(true)
		protected := f.appendStruct(true)

		f.withCache(func(cc *keeperlib.CurrentContext) {
			cc.GetStruct(defender.Id).DestroyAndCommit()
		})

		_, err := f.ms.StructDefenseSet(f.ctx, &types.MsgStructDefenseSet{
			Creator:           f.player.Creator,
			DefenderStructId:  defender.Id,
			ProtectedStructId: protected.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "destroyed")

		_, err = f.ms.StructDefenseClear(f.ctx, &types.MsgStructDefenseClear{
			Creator:          f.player.Creator,
			DefenderStructId: defender.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "destroyed")
	})

	t.Run("generator infuse moves no coins", func(t *testing.T) {
		f := setupDestroyedStructFixture(t)

		generatorType := types.StructType{
			Id:              2,
			Type:            "Small Generator",
			Category:        types.ObjectType_planet,
			MaxHealth:       100,
			PowerGeneration: types.TechPowerGeneration_smallGenerator,
		}
		f.k.SetStructType(f.ctx, generatorType)

		structure := f.appendStruct(true)
		stored, found := f.k.GetStruct(f.ctx, structure.Id)
		require.True(t, found)
		stored.Type = generatorType.Id
		f.k.SetStruct(f.ctx, stored)

		playerAcc, err := sdk.AccAddressFromBech32(f.player.Creator)
		require.NoError(t, err)
		coins := sdk.NewCoins(sdk.NewCoin("ualpha", math.NewInt(1000)))
		require.NoError(t, f.k.BankKeeper().MintCoins(f.ctx, types.ModuleName, coins))
		require.NoError(t, f.k.BankKeeper().SendCoinsFromModuleToAccount(f.ctx, types.ModuleName, playerAcc, coins))

		f.withCache(func(cc *keeperlib.CurrentContext) {
			cc.GetStruct(structure.Id).DestroyAndCommit()
		})
		// The generator has to look online, or the pre-existing offline check would
		// reject the message for the wrong reason and the destroyed guard would go
		// untested.
		testSetStructAttributeFlagAdd(f.k, f.ctx, f.statusAttr(structure.Id), uint64(types.StructStateOnline))

		balanceBefore := f.k.BankKeeper().SpendableCoins(f.ctx, playerAcc)

		_, err = f.ms.StructGeneratorInfuse(f.ctx, &types.MsgStructGeneratorInfuse{
			Creator:      f.player.Creator,
			StructId:     structure.Id,
			InfuseAmount: "1000ualpha",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "destroyed")
		require.Equal(t, balanceBefore, f.k.BankKeeper().SpendableCoins(f.ctx, playerAcc),
			"coins moved into a generator queued for deletion")
	})
}
