package v0_21_0_test

import (
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/app/upgrades"
	v0_21_0 "structs/app/upgrades/v0_21_0"
	keepertest "structs/testutil/keeper"
	structskeeper "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// TestMigrateGuildBankFees verifies the backfill leaves valid (non-nil) zero
// fees on guilds that lack them, preserves guilds that already have fees set,
// and is idempotent.
func TestMigrateGuildBankFees(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	// A guild that already carries an explicit fee must be preserved.
	withFee := types.CreateEmptyGuild()
	withFee.Id = "4-0"
	withFee.Index = 0
	withFee.BankConvertInFee = math.LegacyMustNewDecFromStr("0.25")
	withFee.BankConvertOutFee = math.LegacyMustNewDecFromStr("0.5")
	k.SetGuild(ctx, withFee)

	// Pre-v0.21 protobuf bytes containing only Guild.id. The absent fee fields
	// decode as nil LegacyDec values and cannot be produced via current SetGuild.
	legacyID := "4-1"
	legacyBytes := append([]byte{0x0a, byte(len(legacyID))}, []byte(legacyID)...)
	keepertest.WriteRawGuild(t, ctx, legacyID, legacyBytes)

	before, found := k.GetGuild(ctx, legacyID)
	require.True(t, found)
	require.True(t, before.BankConvertInFee.IsNil())
	require.True(t, before.BankConvertOutFee.IsNil())

	require.NoError(t, v0_21_0.MigrateGuildBankFees(ctx, keepers))

	got0, found := k.GetGuild(ctx, "4-0")
	require.True(t, found)
	require.Equal(t, "0.250000000000000000", got0.BankConvertInFee.String(), "explicit fee preserved")
	require.Equal(t, "0.500000000000000000", got0.BankConvertOutFee.String())

	got1, found := k.GetGuild(ctx, legacyID)
	require.True(t, found)
	require.False(t, got1.BankConvertInFee.IsNil())
	require.False(t, got1.BankConvertOutFee.IsNil())
	require.True(t, got1.BankConvertInFee.IsZero())
	require.True(t, got1.BankConvertOutFee.IsZero())

	// Idempotent: a re-run produces identical values.
	require.NoError(t, v0_21_0.MigrateGuildBankFees(ctx, keepers))
	got0b, _ := k.GetGuild(ctx, "4-0")
	require.Equal(t, got0.BankConvertInFee.String(), got0b.BankConvertInFee.String())
}

// TestMigrateStructTypes verifies the struct-type rewrite populates the new
// canDefend flag: true for fleet types, false for planetary types.
func TestMigrateStructTypes(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	require.NoError(t, v0_21_0.MigrateStructTypes(ctx, keepers))

	// Id 1 (Command Ship) is a fleet type; Id 14 (Ore Extractor) is planetary.
	fleetType, found := k.GetStructType(ctx, 1)
	require.True(t, found)
	require.Equal(t, types.ObjectType_fleet, fleetType.Category)
	require.True(t, fleetType.CanDefend, "fleet struct types must be able to defend")

	planetaryType, found := k.GetStructType(ctx, 14)
	require.True(t, found)
	require.Equal(t, types.ObjectType_planet, planetaryType.Category)
	require.False(t, planetaryType.CanDefend, "planetary struct types must not be able to defend")
}

// TestMigrateDefenderCanDefend verifies the prune removes defender
// registrations whose defending struct can no longer defend (planetary types)
// while retaining valid fleet defenders, and that it emits the same events a
// StructDefenseClear transaction would so the indexer stays in sync.
func TestMigrateDefenderCanDefend(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	k.SetStructType(ctx, types.StructType{Id: 1, Category: types.ObjectType_fleet, CanDefend: true})
	k.SetStructType(ctx, types.StructType{Id: 14, Category: types.ObjectType_planet, CanDefend: false})

	structId := func(index uint64) string {
		return fmt.Sprintf("%d-%d", types.ObjectType_struct, index)
	}

	protectedIndex := uint64(100)
	protectedId := structId(protectedIndex)
	k.SetStruct(ctx, types.Struct{Id: protectedId, Index: protectedIndex, Type: 1})

	fleetDefenderId := structId(101)
	k.SetStruct(ctx, types.Struct{Id: fleetDefenderId, Index: 101, Type: 1})

	planetaryDefenderId := structId(102)
	k.SetStruct(ctx, types.Struct{Id: planetaryDefenderId, Index: 102, Type: 14})

	k.SetStructDefender(ctx, protectedId, protectedIndex, fleetDefenderId)
	k.SetStructDefender(ctx, protectedId, protectedIndex, planetaryDefenderId)

	// Reset the event buffer so we only observe events from the migration.
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithEventManager(sdk.NewEventManager())
	ctx = sdkCtx

	require.NoError(t, v0_21_0.MigrateDefenderCanDefend(ctx, keepers))

	_, fleetFound := k.GetStructDefender(ctx, protectedId, fleetDefenderId)
	require.True(t, fleetFound, "fleet defender should be retained")

	_, planetaryFound := k.GetStructDefender(ctx, protectedId, planetaryDefenderId)
	require.False(t, planetaryFound, "planetary defender should be pruned")

	// The defender's reverse protectedStructIndex attribute must also be cleared.
	planetaryAttrId := structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, planetaryDefenderId)
	require.Equal(t, uint64(0), k.GetStructAttribute(ctx, planetaryAttrId))

	// The prune must emit the same events a StructDefenseClear transaction does.
	var sawDefenderClear, sawAttribute bool
	for _, ev := range sdkCtx.EventManager().Events() {
		if strings.Contains(ev.Type, "EventStructDefenderClear") {
			sawDefenderClear = true
		}
		if strings.Contains(ev.Type, "EventStructAttribute") {
			sawAttribute = true
		}
	}
	require.True(t, sawDefenderClear, "expected EventStructDefenderClear to be emitted")
	require.True(t, sawAttribute, "expected protectedStructIndex EventStructAttribute to be emitted")

	// Idempotent: a re-run finds nothing else to prune and leaves the fleet
	// defender intact.
	require.NoError(t, v0_21_0.MigrateDefenderCanDefend(ctx, keepers))
	_, fleetFound = k.GetStructDefender(ctx, protectedId, fleetDefenderId)
	require.True(t, fleetFound)
}

func TestMigrateOreClocksToPlanet(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(9000)
	ctx = sdkCtx

	k.SetStructType(ctx, types.StructType{
		Id: 14, Category: types.ObjectType_planet,
		PlanetaryMining: types.TechPlanetaryMining_oreMiningRig,
	})
	k.SetStructType(ctx, types.StructType{
		Id: 15, Category: types.ObjectType_planet,
		PlanetaryRefinery: types.TechPlanetaryRefineries_oreRefinery,
	})
	k.SetStructType(ctx, types.StructType{
		Id: 1, Category: types.ObjectType_fleet,
	})

	planetId := fmt.Sprintf("%d-%d", types.ObjectType_planet, 0)
	k.SetPlanet(ctx, types.Planet{Id: planetId})

	onlineStatus := uint64(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline)
	offlineStatus := uint64(types.StructStateMaterialized | types.StructStateBuilt)

	minerA := types.Struct{
		Id: fmt.Sprintf("%d-%d", types.ObjectType_struct, 1), Index: 1,
		Type: 14, LocationId: planetId, LocationType: types.ObjectType_planet,
	}
	minerB := types.Struct{
		Id: fmt.Sprintf("%d-%d", types.ObjectType_struct, 2), Index: 2,
		Type: 14, LocationId: planetId, LocationType: types.ObjectType_planet,
	}
	refinery := types.Struct{
		Id: fmt.Sprintf("%d-%d", types.ObjectType_struct, 3), Index: 3,
		Type: 15, LocationId: planetId, LocationType: types.ObjectType_planet,
	}
	offlineMiner := types.Struct{
		Id: fmt.Sprintf("%d-%d", types.ObjectType_struct, 4), Index: 4,
		Type: 14, LocationId: planetId, LocationType: types.ObjectType_planet,
	}
	fleetStruct := types.Struct{
		Id: fmt.Sprintf("%d-%d", types.ObjectType_struct, 5), Index: 5,
		Type: 1, LocationId: "5-0", LocationType: types.ObjectType_fleet,
	}

	for _, s := range []types.Struct{minerA, minerB, refinery, offlineMiner, fleetStruct} {
		k.SetStruct(ctx, s)
	}

	setStatus := func(id string, status uint64) {
		attrId := structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_status, id)
		k.SetStructAttribute(ctx, attrId, status)
	}
	setStatus(minerA.Id, onlineStatus)
	setStatus(minerB.Id, onlineStatus)
	setStatus(refinery.Id, onlineStatus)
	setStatus(offlineMiner.Id, offlineStatus)
	setStatus(fleetStruct.Id, onlineStatus)

	// minerA clock 100, minerB clock 50 -> planet should get min=50
	k.SetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, minerA.Id), 100)
	k.SetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, minerB.Id), 50)
	k.SetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, offlineMiner.Id), 25)
	k.SetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, refinery.Id), 75)

	require.NoError(t, v0_21_0.MigrateOreClocksToPlanet(ctx, keepers))

	mineQtyId := structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreMiningActiveQuantity, planetId)
	refineQtyId := structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_oreRefiningActiveQuantity, planetId)
	mineClockId := structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreMine, planetId)
	refineClockId := structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetBlockStartOreRefine, planetId)

	require.Equal(t, uint64(2), k.GetPlanetAttribute(ctx, mineQtyId), "two online miners")
	require.Equal(t, uint64(1), k.GetPlanetAttribute(ctx, refineQtyId), "one online refinery")
	require.Equal(t, uint64(50), k.GetPlanetAttribute(ctx, mineClockId), "min of online miner clocks")
	require.Equal(t, uint64(75), k.GetPlanetAttribute(ctx, refineClockId))

	// Old struct clocks cleared.
	require.Equal(t, uint64(0), k.GetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, minerA.Id)))
	require.Equal(t, uint64(0), k.GetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, minerB.Id)))
	require.Equal(t, uint64(0), k.GetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, offlineMiner.Id)))
	require.Equal(t, uint64(0), k.GetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, refinery.Id)))

	// Idempotent: re-run preserves planet clocks and counters.
	require.NoError(t, v0_21_0.MigrateOreClocksToPlanet(ctx, keepers))
	require.Equal(t, uint64(2), k.GetPlanetAttribute(ctx, mineQtyId))
	require.Equal(t, uint64(50), k.GetPlanetAttribute(ctx, mineClockId))
	require.Equal(t, uint64(75), k.GetPlanetAttribute(ctx, refineClockId))
}

func TestMigrateRaiderArrived(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	upgradeHeight := int64(7777)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(upgradeHeight)
	ctx = sdkCtx

	raided := types.Planet{Id: "3-0", LocationListStart: "5-1"}
	idle := types.Planet{Id: "3-1", LocationListStart: ""}
	k.SetPlanet(ctx, raided)
	k.SetPlanet(ctx, idle)

	require.NoError(t, v0_21_0.MigrateRaiderArrived(ctx, keepers))

	raidedAttrId := structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockRaiderArrived, raided.Id)
	idleAttrId := structskeeper.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockRaiderArrived, idle.Id)
	require.Equal(t, uint64(upgradeHeight), k.GetPlanetAttribute(ctx, raidedAttrId))
	require.Equal(t, uint64(0), k.GetPlanetAttribute(ctx, idleAttrId))

	// Idempotent.
	require.NoError(t, v0_21_0.MigrateRaiderArrived(ctx, keepers))
	require.Equal(t, uint64(upgradeHeight), k.GetPlanetAttribute(ctx, raidedAttrId))
}

// TestMigrateJailedReactorEnergy covers the backfill for reactors that were
// already jailed when the upgrade lands. They never passed through the new
// AfterValidatorBeginUnbonding gate, so without this they would keep producing
// energy forever, which is the exploit the release closes.
func TestMigrateJailedReactorEnergy(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)

	// Three reactors: one healthy, one jailed, one whose validator is gone.
	setup := func(seed string) (types.Reactor, sdk.AccAddress, sdk.ValAddress) {
		playerAcc := sdk.AccAddress(fmt.Sprintf("%-36s", seed)[:36])
		player := types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()}
		player.Index = k.GetPlayerCount(ctx)
		player.Id = fmt.Sprintf("%d-%d", types.ObjectType_player, player.Index)
		k.SetPlayer(ctx, player)
		k.SetPlayerCount(ctx, player.Index+1)
		k.SetPlayerIndexForAddress(ctx, player.PrimaryAddress, player.Index)

		valAddr := sdk.ValAddress(playerAcc.Bytes())
		mock.AddValidator(valAddr, math.NewInt(1000))

		reactor := k.AppendReactor(ctx, types.Reactor{
			Validator:         valAddr.String(),
			RawAddress:        valAddr.Bytes(),
			DefaultCommission: math.LegacyZeroDec(),
		})

		k.SetInfusion(ctx, types.Infusion{
			DestinationType: types.ObjectType_reactor,
			DestinationId:   reactor.Id,
			Address:         playerAcc.String(),
			PlayerId:        player.Id,
			Ratio:           types.ReactorFuelToEnergyConversion,
			Fuel:            1000,
			Power:           1000,
			Commission:      math.LegacyZeroDec(),
		})
		k.SetGridAttribute(ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), uint64(1000))

		return reactor, playerAcc, valAddr
	}

	healthyReactor, healthyAcc, _ := setup("migratehealthy")
	jailedReactor, jailedAcc, jailedVal := setup("migratejailed")
	goneReactor, goneAcc, goneVal := setup("migrategone")

	mock.JailValidator(jailedVal)
	mock.RemoveValidator(goneVal)

	require.NoError(t, v0_21_0.MigrateJailedReactorEnergy(ctx, keepers))

	healthy, found := k.GetInfusion(ctx, healthyReactor.Id, healthyAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(types.ReactorFuelToEnergyConversion), healthy.Ratio, "a bonded validator is left alone")
	require.Equal(t, uint64(1000), healthy.Power)

	jailed, found := k.GetInfusion(ctx, jailedReactor.Id, jailedAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(0), jailed.Ratio, "an already-jailed reactor is gated")
	require.Equal(t, uint64(0), jailed.Power)
	require.Equal(t, uint64(1000), jailed.Fuel, "the delegator's stake survives the migration")

	gone, found := k.GetInfusion(ctx, goneReactor.Id, goneAcc.String())
	require.True(t, found)
	require.Equal(t, uint64(0), gone.Ratio, "a missing validator fails closed")
	require.Equal(t, uint64(1000), gone.Fuel)

	// Idempotent: a replayed upgrade block produces identical rows.
	require.NoError(t, v0_21_0.MigrateJailedReactorEnergy(ctx, keepers))

	replayed, found := k.GetInfusion(ctx, jailedReactor.Id, jailedAcc.String())
	require.True(t, found)
	require.Equal(t, jailed, replayed)
}

func TestMigratePrimaryAddressPermissions(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	appendPlayer := func(seed string) types.Player {
		playerAcc := sdk.AccAddress(fmt.Sprintf("%-36s", seed)[:36])
		player := types.Player{
			Creator:        playerAcc.String(),
			PrimaryAddress: playerAcc.String(),
		}
		player.Index = k.GetPlayerCount(ctx)
		player.Id = fmt.Sprintf("%d-%d", types.ObjectType_player, player.Index)
		k.SetPlayer(ctx, player)
		k.SetPlayerCount(ctx, player.Index+1)
		k.SetPlayerIndexForAddress(ctx, player.PrimaryAddress, player.Index)
		return player
	}

	reduced := appendPlayer("migrateprimaryreduced")
	absent := appendPlayer("migrateprimaryabsent")
	alreadyFull := appendPlayer("migrateprimaryfull")
	nonPrimaryOwner := appendPlayer("migrateprimarynonprim")

	reducedId := structskeeper.GetAddressPermissionIDBytes(reduced.PrimaryAddress)
	k.SetPermissionsByBytes(ctx, reducedId, types.PermAll&^types.PermDelete)

	// Absent: leave the primary-address permission record unset.
	absentId := structskeeper.GetAddressPermissionIDBytes(absent.PrimaryAddress)
	require.Equal(t, types.Permissionless, k.GetPermissionsByBytes(ctx, absentId))

	fullId := structskeeper.GetAddressPermissionIDBytes(alreadyFull.PrimaryAddress)
	k.SetPermissionsByBytes(ctx, fullId, types.PermAll)

	// A secondary address associated with the player but not its primary —
	// must not be rewritten by the migration.
	nonPrimaryAcc := sdk.AccAddress(fmt.Sprintf("%-36s", "migrateprimarysecondary")[:36])
	nonPrimaryAddress := nonPrimaryAcc.String()
	k.SetPlayerIndexForAddress(ctx, nonPrimaryAddress, nonPrimaryOwner.Index)
	nonPrimaryId := structskeeper.GetAddressPermissionIDBytes(nonPrimaryAddress)
	k.SetPermissionsByBytes(ctx, nonPrimaryId, types.PermPlay)

	// The non-primary owner's primary address starts full so only the
	// secondary-address assertion is interesting for that player.
	ownerPrimaryId := structskeeper.GetAddressPermissionIDBytes(nonPrimaryOwner.PrimaryAddress)
	k.SetPermissionsByBytes(ctx, ownerPrimaryId, types.PermAll)

	require.NoError(t, v0_21_0.MigratePrimaryAddressPermissions(ctx, keepers))

	require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, reducedId),
		"a reduced primary address is upgraded to PermAll")
	require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, absentId),
		"a missing primary-address permission record is created as PermAll")
	require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, fullId),
		"an already-full primary address is left alone")
	require.Equal(t, types.PermPlay, k.GetPermissionsByBytes(ctx, nonPrimaryId),
		"a non-primary address is not rewritten")

	// Idempotent: a re-run writes nothing further.
	require.NoError(t, v0_21_0.MigratePrimaryAddressPermissions(ctx, keepers))
	require.Equal(t, types.PermAll, k.GetPermissionsByBytes(ctx, reducedId))
	require.Equal(t, types.PermPlay, k.GetPermissionsByBytes(ctx, nonPrimaryId))
}

// TestMigrateFleetQueueLimit plants a 3-deep pre-upgrade raid queue (count
// unset / 0, extra unset / 0) and verifies the head stays while the other
// two visitors are returned home with count seeded to 1.
func TestMigrateFleetQueueLimit(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	appendPlayer := func(seed string) types.Player {
		acc := sdk.AccAddress([]byte(fmt.Sprintf("%-40s", seed)[:40]))
		player := types.Player{Creator: acc.String(), PrimaryAddress: acc.String()}
		player.Index = k.GetPlayerCount(ctx)
		player.Id = fmt.Sprintf("%d-%d", types.ObjectType_player, player.Index)
		k.SetPlayer(ctx, player)
		k.SetPlayerCount(ctx, player.Index+1)
		return player
	}
	appendPlanet := func(owner types.Player) types.Planet {
		planet := types.Planet{
			Id:      fmt.Sprintf("%d-%d", types.ObjectType_planet, k.GetPlanetCount(ctx)),
			Creator: owner.Creator,
			Owner:   owner.Id,
			Status:  types.PlanetStatus_active,
		}
		k.SetPlanet(ctx, planet)
		k.SetPlanetCount(ctx, k.GetPlanetCount(ctx)+1)
		return planet
	}
	appendFleet := func(owner types.Player, homeId string) types.Fleet {
		fleet := types.Fleet{
			Id:           fmt.Sprintf("%d-%s", types.ObjectType_fleet, strings.Split(owner.Id, "-")[1]),
			Owner:        owner.Id,
			LocationId:   homeId,
			LocationType: types.ObjectType_planet,
			Status:       types.FleetStatus_onStation,
		}
		k.SetFleet(ctx, fleet)
		return fleet
	}

	defender := appendPlayer("fql-defender")
	target := appendPlanet(defender)
	defender.PlanetId = target.Id
	k.SetPlayer(ctx, defender)

	a1 := appendPlayer("fql-attacker1")
	h1 := appendPlanet(a1)
	a1.PlanetId = h1.Id
	k.SetPlayer(ctx, a1)
	f1 := appendFleet(a1, h1.Id)

	a2 := appendPlayer("fql-attacker2")
	h2 := appendPlanet(a2)
	a2.PlanetId = h2.Id
	k.SetPlayer(ctx, a2)
	f2 := appendFleet(a2, h2.Id)

	a3 := appendPlayer("fql-attacker3")
	h3 := appendPlanet(a3)
	a3.PlanetId = h3.Id
	k.SetPlayer(ctx, a3)
	f3 := appendFleet(a3, h3.Id)

	// Pre-upgrade deep queue: head f1 -> f2 -> f3 (tail). Count/extra absent (0).
	f1.LocationId = target.Id
	f1.Status = types.FleetStatus_away
	f1.LocationListForward = ""
	f1.LocationListBackward = f2.Id
	k.SetFleet(ctx, f1)

	f2.LocationId = target.Id
	f2.Status = types.FleetStatus_away
	f2.LocationListForward = f1.Id
	f2.LocationListBackward = f3.Id
	k.SetFleet(ctx, f2)

	f3.LocationId = target.Id
	f3.Status = types.FleetStatus_away
	f3.LocationListForward = f2.Id
	f3.LocationListBackward = ""
	k.SetFleet(ctx, f3)

	target.LocationListStart = f1.Id
	target.LocationListLast = f3.Id
	target.LocationListCount = 0
	target.LocationListExtra = 0
	k.SetPlanet(ctx, target)

	require.NoError(t, v0_21_0.MigrateFleetQueueLimit(ctx, keepers))

	gotTarget, found := k.GetPlanet(ctx, target.Id)
	require.True(t, found)
	require.Equal(t, uint64(0), gotTarget.LocationListExtra)
	require.Equal(t, uint64(1), gotTarget.LocationListCount)
	require.Equal(t, f1.Id, gotTarget.LocationListStart)
	require.Equal(t, f1.Id, gotTarget.LocationListLast)

	got1, _ := k.GetFleet(ctx, f1.Id)
	require.Equal(t, target.Id, got1.LocationId)
	require.Equal(t, types.FleetStatus_away, got1.Status)
	require.Equal(t, "", got1.LocationListBackward)

	got2, _ := k.GetFleet(ctx, f2.Id)
	require.Equal(t, h2.Id, got2.LocationId)
	require.Equal(t, types.FleetStatus_onStation, got2.Status)
	require.Equal(t, "", got2.LocationListForward)
	require.Equal(t, "", got2.LocationListBackward)

	got3, _ := k.GetFleet(ctx, f3.Id)
	require.Equal(t, h3.Id, got3.LocationId)
	require.Equal(t, types.FleetStatus_onStation, got3.Status)

	// Idempotent: a second pass leaves the single visitor in place.
	require.NoError(t, v0_21_0.MigrateFleetQueueLimit(ctx, keepers))
	gotTarget, _ = k.GetPlanet(ctx, target.Id)
	require.Equal(t, uint64(1), gotTarget.LocationListCount)
	require.Equal(t, f1.Id, gotTarget.LocationListStart)

	_ = sdkCtx
}

// TestMigrateAgreementCheckpointOverbill verifies the one-block claw-back from
// the provider's earnings pool back into its collateral pool: the amount is one
// block per agreement at that agreement's capacity, it is clamped to what the
// earnings pool actually holds, and providers with nothing owed are untouched.
func TestMigrateAgreementCheckpointOverbill(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	const denom = "ualpha"

	// fund credits an address without debiting anyone, which is the mock's only
	// way to seed a starting balance.
	fund := func(addr sdk.AccAddress, amount int64) {
		require.NoError(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr,
			sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(amount)))))
	}

	// appendProvider stores a provider with a published rate and penalty, plus
	// agreements of the given capacities indexed against it.
	appendProvider := func(id string, rate int64, penalty string, capacities ...uint64) types.Provider {
		provider := types.Provider{
			Id:                          id,
			Rate:                        sdk.NewCoin(denom, math.NewInt(rate)),
			ProviderCancellationPenalty: math.LegacyMustNewDecFromStr(penalty),
			ConsumerCancellationPenalty: math.LegacyMustNewDecFromStr("0"),
		}
		stored, err := k.SetProvider(ctx, provider)
		require.NoError(t, err)

		for i, capacity := range capacities {
			agreement := types.Agreement{
				Id:         fmt.Sprintf("%s-agreement-%d", id, i),
				ProviderId: id,
				Capacity:   capacity,
				StartBlock: 100,
				EndBlock:   200,
			}
			_, err := k.SetAgreement(ctx, agreement)
			require.NoError(t, err)
			require.NoError(t, k.SetAgreementProviderIndex(ctx, id, agreement.Id))
		}

		return stored
	}

	// Two agreements: one block each at capacity 100 and 250, rate 10, half the
	// rate being the non-penalty share the checkpoint swept. 500 + 1250 = 1750.
	funded := appendProvider("provider-funded", 10, "0.5", 100, 250)
	fundedEarnings := structskeeper.GetProviderEarningsPoolLocation(funded.Id)
	fundedCollateral := structskeeper.GetProviderCollateralPoolLocation(funded.Id)
	fund(fundedEarnings, 10000)

	// Already withdrawn all but 300 of a 500 claim, so the claw-back clamps.
	drained := appendProvider("provider-drained", 10, "0.5", 100)
	drainedEarnings := structskeeper.GetProviderEarningsPoolLocation(drained.Id)
	drainedCollateral := structskeeper.GetProviderCollateralPoolLocation(drained.Id)
	fund(drainedEarnings, 300)

	// No agreements, so nothing is owed and nothing should move.
	idle := appendProvider("provider-idle", 10, "0.5")
	idleEarnings := structskeeper.GetProviderEarningsPoolLocation(idle.Id)
	fund(idleEarnings, 7777)

	require.NoError(t, v0_21_0.MigrateAgreementCheckpointOverbill(ctx, keepers))

	balance := func(addr sdk.AccAddress) int64 {
		return k.BankKeeper().SpendableCoin(ctx, addr, denom).Amount.Int64()
	}

	require.Equal(t, int64(1750), balance(fundedCollateral),
		"one block per agreement at its own capacity should come back")
	require.Equal(t, int64(10000-1750), balance(fundedEarnings))

	require.Equal(t, int64(300), balance(drainedCollateral),
		"the claw-back must clamp to what the earnings pool still holds")
	require.Equal(t, int64(0), balance(drainedEarnings))

	require.Equal(t, int64(7777), balance(idleEarnings),
		"a provider with no agreements owes nothing")
	require.Equal(t, int64(0), balance(structskeeper.GetProviderCollateralPoolLocation(idle.Id)))
}

// phantomFixture builds a small world for MigrateStructPhantomAggregates: one
// player, one planet, and struct types covering each aggregate the migration
// rebuilds.
type phantomFixture struct {
	t   *testing.T
	k   structskeeper.Keeper
	ctx sdk.Context

	player types.Player
	planet types.Planet
}

const (
	phantomBunkerType   = 1 // ore reserve defense + defensive cannon
	phantomMinerType    = 2 // ore mining
	phantomGeneratorTyp = 3 // power generation, the source of orphan grid rows
	phantomShield       = 9
	phantomBuildDraw    = 400
	phantomPassiveDraw  = 250
)

func setupPhantomFixture(t *testing.T) *phantomFixture {
	t.Helper()

	k, goCtx := keepertest.StructsKeeper(t)
	ctx := sdk.UnwrapSDKContext(goCtx)

	acc := sdk.AccAddress([]byte(fmt.Sprintf("%-40s", "phantom-owner")[:40]))
	player := types.Player{
		Creator:        acc.String(),
		PrimaryAddress: acc.String(),
		Index:          k.GetPlayerCount(ctx),
	}
	player.Id = fmt.Sprintf("%d-%d", types.ObjectType_player, player.Index)
	k.SetPlayer(ctx, player)
	k.SetPlayerCount(ctx, player.Index+1)

	planet := types.Planet{
		Id:      fmt.Sprintf("%d-%d", types.ObjectType_planet, k.GetPlanetCount(ctx)),
		Creator: player.Creator,
		Owner:   player.Id,
		Status:  types.PlanetStatus_active,
	}
	k.SetPlanet(ctx, planet)
	k.SetPlanetCount(ctx, k.GetPlanetCount(ctx)+1)

	k.SetStructType(ctx, types.StructType{
		Id:                          phantomBunkerType,
		Category:                    types.ObjectType_planet,
		BuildDraw:                   phantomBuildDraw,
		PassiveDraw:                 phantomPassiveDraw,
		OreReserveDefenses:          types.TechOreReserveDefenses_oreBunker,
		PlanetaryShieldContribution: phantomShield,
		PlanetaryDefenses:           types.TechPlanetaryDefenses_defensiveCannon,
	})
	k.SetStructType(ctx, types.StructType{
		Id:              phantomMinerType,
		Category:        types.ObjectType_planet,
		BuildDraw:       phantomBuildDraw,
		PassiveDraw:     phantomPassiveDraw,
		PlanetaryMining: types.TechPlanetaryMining_oreMiningRig,
	})
	k.SetStructType(ctx, types.StructType{
		Id:              phantomGeneratorTyp,
		Category:        types.ObjectType_planet,
		BuildDraw:       phantomBuildDraw,
		PassiveDraw:     phantomPassiveDraw,
		PowerGeneration: types.TechPowerGeneration_smallGenerator,
	})

	return &phantomFixture{t: t, k: k, ctx: ctx, player: player, planet: planet}
}

func (f *phantomFixture) appendStruct(typeId uint64, status types.StructState) types.Struct {
	f.t.Helper()

	index := f.k.GetStructCount(f.ctx)
	structure := types.Struct{
		Id:           fmt.Sprintf("%d-%d", types.ObjectType_struct, index),
		Index:        index,
		Creator:      f.player.Creator,
		Owner:        f.player.Id,
		Type:         typeId,
		LocationId:   f.planet.Id,
		LocationType: types.ObjectType_planet,
	}
	f.k.SetStruct(f.ctx, structure)
	f.k.SetStructCount(f.ctx, index+1)
	f.k.SetStructAttribute(f.ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id), uint64(status))
	return structure
}

func (f *phantomFixture) planetAttr(attributeType types.PlanetAttributeType) uint64 {
	return f.k.GetPlanetAttribute(f.ctx, structskeeper.GetPlanetAttributeIDByObjectId(attributeType, f.planet.Id))
}

func (f *phantomFixture) setPlanetAttr(attributeType types.PlanetAttributeType, value uint64) {
	f.k.SetPlanetAttribute(f.ctx, structskeeper.GetPlanetAttributeIDByObjectId(attributeType, f.planet.Id), value)
}

func (f *phantomFixture) structsLoad() uint64 {
	return f.k.GetGridAttribute(f.ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, f.player.Id))
}

func (f *phantomFixture) typeCount(typeId uint64) uint64 {
	return f.k.GetStructAttribute(f.ctx, structskeeper.GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, f.player.Id, typeId))
}

// seedBaseAggregates writes the values NewPlayer and NewPlanet establish before
// any struct exists.
func (f *phantomFixture) seedBaseAggregates() {
	f.k.SetGridAttribute(f.ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, f.player.Id), types.PlayerPassiveDraw)
	f.setPlanetAttr(types.PlanetAttributeType_planetaryShield, types.PlanetaryShieldBase)
}

// buildThroughRuntime drives the same cache calls the handlers do, so the
// aggregates come out of the production code rather than out of the test's idea
// of it: InitiateStruct reserves BuildDraw and bumps the type count,
// StructBuildComplete releases the reservation and goes online, StructDeactivate
// goes back offline.
func (f *phantomFixture) buildThroughRuntime(typeId uint64, complete bool, online bool) types.Struct {
	f.t.Helper()

	structure := f.appendStruct(typeId, types.StructState(types.StructStateMaterialized))

	cc := f.k.NewCurrentContext(f.ctx)
	cache := cc.GetStruct(structure.Id)
	structType, found := cc.GetStructType(typeId)
	require.True(f.t, found)

	cache.GetOwner().StructsLoadIncrement(structType.GetStructType().BuildDraw)
	cache.GetOwner().BuildQuantityIncrement(typeId)

	if complete {
		cache.GetOwner().StructsLoadDecrement(structType.GetStructType().BuildDraw)
		cache.StatusAddBuilt()
		cache.GoOnline()
		if !online {
			cache.GoOffline()
		}
	}

	cc.CommitAll()
	return structure
}

// TestMigrateStructPhantomAggregates_RepairsCorruption plants the exact damage
// the two bugs produced — a load and type count released twice, and a planetary
// shield, cannon and ore rig left behind by a struct that was reactivated after
// destruction and then swept — and checks each one is rebuilt from the structs
// still standing.
func TestMigrateStructPhantomAggregates_RepairsCorruption(t *testing.T) {
	f := setupPhantomFixture(t)
	keepers := &upgrades.Keepers{StructsKeeper: f.k}

	online := types.StructState(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline)
	building := types.StructState(types.StructStateMaterialized)
	destroyed := types.StructState(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateDestroyed)

	// What actually survives: an online bunker, an online miner, one struct still
	// building, and a destroyed one awaiting the sweep that contributes nothing.
	f.appendStruct(phantomBunkerType, online)
	f.appendStruct(phantomMinerType, online)
	f.appendStruct(phantomBunkerType, building)
	f.appendStruct(phantomBunkerType, destroyed)

	// The damage. Load and type count were each released one extra time by the
	// replayed destruction; the planet carries a shield, a cannon and an ore rig
	// from a struct that no longer exists.
	f.k.SetGridAttribute(f.ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, f.player.Id),
		types.PlayerPassiveDraw+phantomPassiveDraw*2+phantomBuildDraw-phantomBuildDraw)
	f.k.SetStructAttribute(f.ctx, structskeeper.GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, f.player.Id, phantomBunkerType), 1)
	f.k.SetStructAttribute(f.ctx, structskeeper.GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, f.player.Id, phantomMinerType), 1)
	f.setPlanetAttr(types.PlanetAttributeType_planetaryShield, types.PlanetaryShieldBase+phantomShield*2)
	f.setPlanetAttr(types.PlanetAttributeType_defensiveCannonQuantity, 2)
	f.setPlanetAttr(types.PlanetAttributeType_oreMiningActiveQuantity, 2)

	// Grid rows belonging to a generator the sweep already removed.
	ghostId := fmt.Sprintf("%d-%d", types.ObjectType_struct, 999)
	for _, attributeType := range []types.GridAttributeType{
		types.GridAttributeType_ready,
		types.GridAttributeType_load,
		types.GridAttributeType_capacity,
		types.GridAttributeType_fuel,
		types.GridAttributeType_power,
	} {
		f.k.SetGridAttribute(f.ctx, structskeeper.GetGridAttributeIDByObjectId(attributeType, ghostId), 42)
	}

	require.NoError(t, v0_21_0.MigrateStructPhantomAggregates(f.ctx, keepers))

	require.Equal(t, uint64(types.PlayerPassiveDraw+phantomPassiveDraw*2+phantomBuildDraw), f.structsLoad(),
		"load must come back to the base draw plus what the surviving structs actually take")
	require.Equal(t, uint64(2), f.typeCount(phantomBunkerType),
		"the online and building bunkers both count; the destroyed one does not")
	require.Equal(t, uint64(1), f.typeCount(phantomMinerType))

	require.Equal(t, uint64(types.PlanetaryShieldBase+phantomShield), f.planetAttr(types.PlanetAttributeType_planetaryShield),
		"only the one online bunker still contributes shield")
	require.Equal(t, uint64(1), f.planetAttr(types.PlanetAttributeType_defensiveCannonQuantity))
	require.Equal(t, uint64(1), f.planetAttr(types.PlanetAttributeType_oreMiningActiveQuantity))

	for _, attributeType := range []types.GridAttributeType{
		types.GridAttributeType_ready,
		types.GridAttributeType_load,
		types.GridAttributeType_capacity,
		types.GridAttributeType_fuel,
		types.GridAttributeType_power,
	} {
		require.Equal(t, uint64(0),
			f.k.GetGridAttribute(f.ctx, structskeeper.GetGridAttributeIDByObjectId(attributeType, ghostId)),
			"grid rows keyed to a struct that no longer exists must be cleared")
	}

	// Idempotent: every value is derived and assigned, so a re-run is a no-op.
	require.NoError(t, v0_21_0.MigrateStructPhantomAggregates(f.ctx, keepers))
	require.Equal(t, uint64(types.PlayerPassiveDraw+phantomPassiveDraw*2+phantomBuildDraw), f.structsLoad())
	require.Equal(t, uint64(types.PlanetaryShieldBase+phantomShield), f.planetAttr(types.PlanetAttributeType_planetaryShield))
	require.Equal(t, uint64(2), f.typeCount(phantomBunkerType))
}

// TestMigrateStructPhantomAggregates_LeavesHealthyStateAlone is the important
// one. A recompute that disagrees with the runtime does more damage than the bug
// it repairs, so this builds its world by driving the production cache calls and
// then asserts the migration changes nothing at all.
func TestMigrateStructPhantomAggregates_LeavesHealthyStateAlone(t *testing.T) {
	f := setupPhantomFixture(t)
	keepers := &upgrades.Keepers{StructsKeeper: f.k}

	f.seedBaseAggregates()

	f.buildThroughRuntime(phantomBunkerType, true, true)  // built and online
	f.buildThroughRuntime(phantomBunkerType, true, false) // built, then deactivated
	f.buildThroughRuntime(phantomMinerType, true, true)
	f.buildThroughRuntime(phantomGeneratorTyp, false, false) // still building

	gridBefore := f.k.GetAllGridExport(f.ctx)
	planetBefore := f.k.GetAllPlanetAttributeExport(f.ctx)
	structBefore := f.k.GetAllStructAttributeExport(f.ctx)

	require.NoError(t, v0_21_0.MigrateStructPhantomAggregates(f.ctx, keepers))

	require.Equal(t, gridBefore, f.k.GetAllGridExport(f.ctx),
		"the recompute disagrees with the runtime somewhere in the grid store")
	require.Equal(t, planetBefore, f.k.GetAllPlanetAttributeExport(f.ctx),
		"the recompute disagrees with the runtime on a planet aggregate")
	require.Equal(t, structBefore, f.k.GetAllStructAttributeExport(f.ctx),
		"the recompute disagrees with the runtime on a type count")

	// Spelled out, so a failure above says which number is wrong rather than only
	// that something is.
	require.Equal(t, uint64(types.PlayerPassiveDraw+phantomPassiveDraw*2+phantomBuildDraw), f.structsLoad())
	require.Equal(t, uint64(types.PlanetaryShieldBase+phantomShield), f.planetAttr(types.PlanetAttributeType_planetaryShield))
	require.Equal(t, uint64(1), f.planetAttr(types.PlanetAttributeType_defensiveCannonQuantity))
	require.Equal(t, uint64(0), f.planetAttr(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity))
	require.Equal(t, uint64(1), f.planetAttr(types.PlanetAttributeType_oreMiningActiveQuantity))
	require.Equal(t, uint64(0), f.planetAttr(types.PlanetAttributeType_oreRefiningActiveQuantity))
	require.Equal(t, uint64(2), f.typeCount(phantomBunkerType))
	require.Equal(t, uint64(1), f.typeCount(phantomMinerType))
	require.Equal(t, uint64(1), f.typeCount(phantomGeneratorTyp))
}

// TestMigrateStructPhantomAggregates_FleetStructsFollowTheirFleet pins the one
// piece of location logic the recompute has to mirror: a struct in a fleet
// contributes to whatever planet the fleet is visiting, not to a planet of its
// own, which is how StructCache.GetPlanet resolves it.
func TestMigrateStructPhantomAggregates_FleetStructsFollowTheirFleet(t *testing.T) {
	f := setupPhantomFixture(t)
	keepers := &upgrades.Keepers{StructsKeeper: f.k}

	fleetType := uint64(4)
	f.k.SetStructType(f.ctx, types.StructType{
		Id:                          fleetType,
		Category:                    types.ObjectType_fleet,
		PassiveDraw:                 phantomPassiveDraw,
		OreReserveDefenses:          types.TechOreReserveDefenses_oreBunker,
		PlanetaryShieldContribution: phantomShield,
	})

	fleet := types.Fleet{
		Id:           fmt.Sprintf("%d-%d", types.ObjectType_fleet, f.player.Index),
		Owner:        f.player.Id,
		LocationId:   f.planet.Id,
		LocationType: types.ObjectType_planet,
		Status:       types.FleetStatus_onStation,
	}
	f.k.SetFleet(f.ctx, fleet)

	index := f.k.GetStructCount(f.ctx)
	structure := types.Struct{
		Id:           fmt.Sprintf("%d-%d", types.ObjectType_struct, index),
		Index:        index,
		Creator:      f.player.Creator,
		Owner:        f.player.Id,
		Type:         fleetType,
		LocationId:   fleet.Id,
		LocationType: types.ObjectType_fleet,
	}
	f.k.SetStruct(f.ctx, structure)
	f.k.SetStructCount(f.ctx, index+1)
	f.k.SetStructAttribute(f.ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id),
		uint64(types.StructStateMaterialized|types.StructStateBuilt|types.StructStateOnline))

	f.setPlanetAttr(types.PlanetAttributeType_planetaryShield, types.PlanetaryShieldBase)

	require.NoError(t, v0_21_0.MigrateStructPhantomAggregates(f.ctx, keepers))

	require.Equal(t, uint64(types.PlanetaryShieldBase+phantomShield), f.planetAttr(types.PlanetAttributeType_planetaryShield),
		"a fleet struct's shield belongs to the planet its fleet is at")
	require.Equal(t, uint64(types.PlayerPassiveDraw+phantomPassiveDraw), f.structsLoad())
}
