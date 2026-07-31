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
