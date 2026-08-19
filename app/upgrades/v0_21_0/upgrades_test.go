package v0_21_0_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"structs/app/upgrades"
	v0_21_0 "structs/app/upgrades/v0_21_0"
	keepertest "structs/testutil/keeper"
	structskeeper "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMigrateProviderPoolAddressIndex(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	provider := types.Provider{Id: "10-44", Index: 44}
	k.ImportProvider(ctx, provider)

	collateral := structskeeper.GetProviderCollateralPoolLocation(provider.Id).String()
	earnings := structskeeper.GetProviderEarningsPoolLocation(provider.Id).String()
	_, _, found := k.GetProviderPoolAddress(ctx, collateral)
	require.False(t, found)

	v0_21_0.MigrateProviderPoolAddressIndex(ctx, keepers)

	gotProvider, kind, found := k.GetProviderPoolAddress(ctx, collateral)
	require.True(t, found)
	require.Equal(t, provider.Id, gotProvider)
	require.Equal(t, structskeeper.ProviderPoolKindCollateral, kind)

	gotProvider, kind, found = k.GetProviderPoolAddress(ctx, earnings)
	require.True(t, found)
	require.Equal(t, provider.Id, gotProvider)
	require.Equal(t, structskeeper.ProviderPoolKindEarnings, kind)
}

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

// TestMigrateGuildJoinBypassLevels verifies that guilds carrying a bypass level
// outside the declared enum are clamped to closed and that guilds holding
// declared levels are left exactly as they were.
//
// Records like these were writable before v0.21.0: the update handlers passed
// msg.GuildJoinBypassLevel straight through and proto3 enums are open, so any
// int32 persisted. SetGuild has no validation, which is what lets this test
// stage the state at all.
func TestMigrateGuildJoinBypassLevels(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	poisoned := types.CreateEmptyGuild()
	poisoned.Id = "4-0"
	poisoned.Index = 0
	poisoned.JoinInfusionMinimumBypassByRequest = types.GuildJoinBypassLevel(500)
	poisoned.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel(500)
	k.SetGuild(ctx, poisoned)

	// Only one field poisoned: the untouched field must survive the clamp.
	halfPoisoned := types.CreateEmptyGuild()
	halfPoisoned.Id = "4-1"
	halfPoisoned.Index = 1
	halfPoisoned.JoinInfusionMinimumBypassByRequest = types.GuildJoinBypassLevel(-1)
	halfPoisoned.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_member
	k.SetGuild(ctx, halfPoisoned)

	healthy := types.CreateEmptyGuild()
	healthy.Id = "4-2"
	healthy.Index = 2
	healthy.JoinInfusionMinimumBypassByRequest = types.GuildJoinBypassLevel_member
	healthy.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_permissioned
	k.SetGuild(ctx, healthy)

	zeroRank := types.CreateEmptyGuild()
	zeroRank.Id = "4-3"
	zeroRank.Index = 3
	zeroRank.EntryRank = 0
	k.SetGuild(ctx, zeroRank)

	require.NoError(t, v0_21_0.MigrateGuildJoinBypassLevels(ctx, keepers))

	got0, found := k.GetGuild(ctx, "4-0")
	require.True(t, found)
	require.Equal(t, types.GuildJoinBypassLevel_closed, got0.JoinInfusionMinimumBypassByRequest)
	require.Equal(t, types.GuildJoinBypassLevel_closed, got0.JoinInfusionMinimumBypassByInvite)

	got1, found := k.GetGuild(ctx, "4-1")
	require.True(t, found)
	require.Equal(t, types.GuildJoinBypassLevel_closed, got1.JoinInfusionMinimumBypassByRequest)
	require.Equal(t, types.GuildJoinBypassLevel_member, got1.JoinInfusionMinimumBypassByInvite, "a declared level must not be clamped")

	got2, found := k.GetGuild(ctx, "4-2")
	require.True(t, found)
	require.Equal(t, types.GuildJoinBypassLevel_member, got2.JoinInfusionMinimumBypassByRequest)
	require.Equal(t, types.GuildJoinBypassLevel_permissioned, got2.JoinInfusionMinimumBypassByInvite)

	got3, found := k.GetGuild(ctx, "4-3")
	require.True(t, found)
	require.Equal(t, uint64(types.DefaultEntryRank), got3.EntryRank)

	// Idempotent: closed is a declared value, so a re-run finds nothing to do.
	require.NoError(t, v0_21_0.MigrateGuildJoinBypassLevels(ctx, keepers))
	got0b, _ := k.GetGuild(ctx, "4-0")
	require.Equal(t, got0.JoinInfusionMinimumBypassByRequest, got0b.JoinInfusionMinimumBypassByRequest)
	require.Equal(t, got0.JoinInfusionMinimumBypassByInvite, got0b.JoinInfusionMinimumBypassByInvite)
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

	unknownTypeDefenderId := structId(103)
	k.SetStruct(ctx, types.Struct{Id: unknownTypeDefenderId, Index: 103, Type: 999})

	k.SetStructDefender(ctx, protectedId, protectedIndex, fleetDefenderId)
	k.SetStructDefender(ctx, protectedId, protectedIndex, planetaryDefenderId)
	k.SetStructDefender(ctx, protectedId, protectedIndex, unknownTypeDefenderId)

	// Reset the event buffer so we only observe events from the migration.
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithEventManager(sdk.NewEventManager())
	ctx = sdkCtx

	require.NoError(t, v0_21_0.MigrateDefenderCanDefend(ctx, keepers))

	_, fleetFound := k.GetStructDefender(ctx, protectedId, fleetDefenderId)
	require.True(t, fleetFound, "fleet defender should be retained")

	_, planetaryFound := k.GetStructDefender(ctx, protectedId, planetaryDefenderId)
	require.False(t, planetaryFound, "planetary defender should be pruned")

	_, unknownFound := k.GetStructDefender(ctx, protectedId, unknownTypeDefenderId)
	require.False(t, unknownFound, "an unknown type cannot be trusted with defender authority")

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
	destroyedMiner := types.Struct{
		Id: fmt.Sprintf("%d-%d", types.ObjectType_struct, 6), Index: 6,
		Type: 14, LocationId: planetId, LocationType: types.ObjectType_planet,
	}
	fleetStruct := types.Struct{
		Id: fmt.Sprintf("%d-%d", types.ObjectType_struct, 5), Index: 5,
		Type: 1, LocationId: "5-0", LocationType: types.ObjectType_fleet,
	}

	for _, s := range []types.Struct{minerA, minerB, refinery, offlineMiner, destroyedMiner, fleetStruct} {
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
	setStatus(destroyedMiner.Id, onlineStatus|uint64(types.StructStateDestroyed))
	setStatus(fleetStruct.Id, onlineStatus)

	// minerA clock 100, minerB clock 50 -> planet should get min=50
	k.SetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, minerA.Id), 100)
	k.SetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, minerB.Id), 50)
	k.SetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, offlineMiner.Id), 25)
	k.SetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, destroyedMiner.Id), 10)
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
	require.Equal(t, uint64(0), k.GetStructAttribute(ctx, structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, destroyedMiner.Id)))
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

/* All three writes MigrateGuildCharter makes are the difference between the
 * charter working and being permanently unreachable, so each is asserted rather
 * than the migration merely being run.
 */
func TestMigrateGuildCharter(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	keepers := &upgrades.Keepers{StructsKeeper: k}

	upgradeHeight := int64(9100)
	ctx = ctx.WithBlockHeight(upgradeHeight)

	// A Params record written by an earlier binary decodes both new fields as
	// zero, which is what the migration has to repair.
	params := k.GetParams(ctx)
	params.GuildCharterDifficultyRange = 0
	params.GuildCharterReactorAge = 0
	require.NoError(t, k.SetParams(ctx, params))

	freshVal := sdk.ValAddress([]byte("charter-fresh-------"))
	fresh := k.AppendReactor(ctx, types.Reactor{
		Validator:         freshVal.String(),
		RawAddress:        freshVal.Bytes(),
		DefaultCommission: math.LegacyZeroDec(),
	})
	spentVal := sdk.ValAddress([]byte("charter-spent-------"))
	spent := k.AppendReactor(ctx, types.Reactor{
		Validator:         spentVal.String(),
		RawAddress:        spentVal.Bytes(),
		DefaultCommission: math.LegacyZeroDec(),
	})

	// AppendReactor stamps new reactors itself, so clear both to stand in for
	// records written before the field existed.
	for _, reactor := range []types.Reactor{fresh, spent} {
		reactor.GuildCharterEligibleHeight = 0
		if reactor.Id == spent.Id {
			reactor.GuildId = "4-7"
		}
		k.SetReactor(ctx, reactor)
	}

	require.NoError(t, v0_21_0.MigrateGuildCharter(ctx, keepers))

	/* Zero is the one difficulty range CalculateDifficulty cannot take: it pins
	 * the requirement at 64 leading zeros forever, so leaving it would make
	 * founding a guild by proof permanently impossible.
	 */
	migrated := k.GetParams(ctx)
	require.Equal(t, uint64(types.DefaultGuildCharterDifficultyRange), migrated.GuildCharterDifficultyRange)
	require.Equal(t, uint64(types.DefaultGuildCharterReactorAge), migrated.GuildCharterReactorAge)

	anchor, found := k.GetGuildCharterAnchor(ctx)
	require.True(t, found, "the anchor cannot be derived from anything else on disk")
	require.Equal(t, uint64(upgradeHeight), anchor)
	require.Equal(t, uint64(0), k.CharterAge(ctx))
	require.Equal(t, 64, k.CharterDifficulty(ctx), "the puzzle starts at its hardest")

	/* Eligibility is a stored height and zero reads as never eligible, so an
	 * unstamped reactor would be shut out of the free path for good.
	 */
	stampedFresh, foundFresh := k.GetReactor(ctx, fresh.Id)
	require.True(t, foundFresh)
	require.Equal(t, uint64(upgradeHeight), stampedFresh.GuildCharterEligibleHeight)

	// A reactor that already founded a guild is stamped too, but its GuildId is
	// what spends the entitlement, so the stamp buys it nothing.
	stampedSpent, foundSpent := k.GetReactor(ctx, spent.Id)
	require.True(t, foundSpent)
	require.Equal(t, uint64(upgradeHeight), stampedSpent.GuildCharterEligibleHeight)
	require.Equal(t, "4-7", stampedSpent.GuildId)

	// Idempotent, including the params of a chain that has already tuned them.
	tuned := k.GetParams(ctx)
	tuned.GuildCharterDifficultyRange = 4000
	tuned.GuildCharterReactorAge = 50
	require.NoError(t, k.SetParams(ctx, tuned))

	require.NoError(t, v0_21_0.MigrateGuildCharter(ctx, keepers))

	replayed := k.GetParams(ctx)
	require.Equal(t, uint64(4000), replayed.GuildCharterDifficultyRange, "a tuned range must survive a replay")
	require.Equal(t, uint64(50), replayed.GuildCharterReactorAge)

	replayedFresh, found := k.GetReactor(ctx, fresh.Id)
	require.True(t, found)
	require.Equal(t, stampedFresh, replayedFresh)
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

	gotTarget.LocationListStart = "2-missing"
	gotTarget.LocationListCount = 7
	k.SetPlanet(ctx, gotTarget)
	require.NoError(t, v0_21_0.MigrateFleetQueueLimit(ctx, keepers))
	gotTarget, _ = k.GetPlanet(ctx, target.Id)
	require.Equal(t, uint64(7), gotTarget.LocationListCount,
		"a partial walk must not replace the stored count with zero")
	require.Equal(t, "2-missing", gotTarget.LocationListStart)

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

func TestMigrateStructPhantomAggregates_DestroyedOnlineStaysRepairedAfterSweep(t *testing.T) {
	f := setupPhantomFixture(t)
	keepers := &upgrades.Keepers{StructsKeeper: f.k}
	f.seedBaseAggregates()

	structure := f.buildThroughRuntime(phantomBunkerType, true, true)
	statusAttr := structskeeper.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id)
	status := f.k.GetStructAttribute(f.ctx, statusAttr)
	f.k.SetStructAttribute(f.ctx, statusAttr, status|uint64(types.StructStateDestroyed))
	f.k.AppendStructDestructionQueue(f.ctx, structure.Id)

	require.NoError(t, v0_21_0.MigrateStructPhantomAggregates(f.ctx, keepers))
	require.Equal(t, uint64(types.PlayerPassiveDraw), f.structsLoad())
	require.Equal(t, uint64(types.PlanetaryShieldBase), f.planetAttr(types.PlanetAttributeType_planetaryShield))
	require.Equal(t, uint64(0), f.planetAttr(types.PlanetAttributeType_defensiveCannonQuantity))

	sweepCtx := f.ctx.WithBlockHeight(f.ctx.BlockHeight() + types.StructSweepDelay)
	f.k.StructSweepDestroyed(sweepCtx)

	require.Equal(t, uint64(types.PlayerPassiveDraw), f.structsLoad(),
		"the sweep must not subtract a struct the migration already excluded")
	require.Equal(t, uint64(types.PlanetaryShieldBase), f.planetAttr(types.PlanetAttributeType_planetaryShield))
	require.Equal(t, uint64(0), f.planetAttr(types.PlanetAttributeType_defensiveCannonQuantity))
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

// guildNameFixture stores guilds the way GuildUpdateName does: the name on the
// guild record and a matching index row. Anything that deviates from that shape
// is written by the individual tests.
type guildNameFixture struct {
	t   *testing.T
	k   structskeeper.Keeper
	ctx sdk.Context
}

func newGuildNameFixture(t *testing.T) *guildNameFixture {
	t.Helper()
	k, ctx := keepertest.StructsKeeper(t)
	return &guildNameFixture{t: t, k: k, ctx: sdk.UnwrapSDKContext(ctx)}
}

func (f *guildNameFixture) keepers() *upgrades.Keepers {
	return &upgrades.Keepers{StructsKeeper: f.k}
}

// named stores a guild carrying name with its index row, as a rename would.
func (f *guildNameFixture) named(index uint64, name string) types.Guild {
	f.t.Helper()
	guild := types.CreateEmptyGuild()
	guild.Index = index
	guild.Id = fmt.Sprintf("%d-%d", types.ObjectType_guild, index)
	guild.Name = name
	f.k.SetGuild(f.ctx, guild)
	if name != "" {
		f.k.SetGuildNameIndex(f.ctx, name, guild.Id)
	}
	return guild
}

func (f *guildNameFixture) name(guildId string) string {
	f.t.Helper()
	guild, found := f.k.GetGuild(f.ctx, guildId)
	require.True(f.t, found, "guild %s vanished", guildId)
	return guild.Name
}

// TestMigrateGuildNameIndex_HealthyStateIsUnchanged is the assertion that
// matters most, because a no-op is what the rebuild is supposed to be.
//
// The new NormalizeName folds case and trims space through checked-in Unicode
// 15.0.0 tables instead of the compiling toolchain's, and every Go release the
// chain has run ships exactly that version, so the key bytes are identical.
// This shows it end to end at the store level; TestNormalizeNameMatchesLegacyForm
// in x/structs/types proves the same thing exhaustively over every code point.
func TestMigrateGuildNameIndex_HealthyStateIsUnchanged(t *testing.T) {
	f := newGuildNameFixture(t)

	ascii := f.named(0, "Alpha Guild")
	unicodeName := f.named(1, "Ñoño Collective")
	f.named(2, "beta-guild_7")
	nameless := f.named(3, "") // created but never renamed, which is most guilds

	before := f.k.GetAllGuildNameIndex(f.ctx)
	require.Len(t, before, 3, "only the named guilds should hold a row")

	require.NoError(t, v0_21_0.MigrateGuildNameIndex(f.ctx, f.keepers()))

	require.Equal(t, before, f.k.GetAllGuildNameIndex(f.ctx),
		"the rebuild re-keyed the guild name index; NormalizeName changed its output")
	require.Equal(t, "Alpha Guild", f.name(ascii.Id), "a valid name must survive untouched")
	require.Equal(t, "Ñoño Collective", f.name(unicodeName.Id),
		"a non-ASCII name is exactly what a case-folding change would have moved")
	require.Equal(t, "", f.name(nameless.Id))

	// Idempotent: the second run clears what the first wrote and rebuilds it.
	require.NoError(t, v0_21_0.MigrateGuildNameIndex(f.ctx, f.keepers()))
	require.Equal(t, before, f.k.GetAllGuildNameIndex(f.ctx))
}

// TestMigrateGuildNameIndex_ClearsOrphanedRows covers the reason the rebuild
// clears the prefix rather than overwriting row by row. RemoveGuildNameIndex
// deletes by re-normalizing a name, so a row no name maps to is unreachable
// through the ordinary accessors and would sit in state forever, refusing that
// name to every guild.
func TestMigrateGuildNameIndex_ClearsOrphanedRows(t *testing.T) {
	f := newGuildNameFixture(t)

	kept := f.named(0, "Alpha Guild")

	// A row for a guild that no longer carries that name.
	f.k.SetGuildNameIndex(f.ctx, "Stale Name", kept.Id)
	// A row pointing at a guild that does not exist at all.
	f.k.SetGuildNameIndex(f.ctx, "Ghost Guild", "4-999")
	require.Len(t, f.k.GetAllGuildNameIndex(f.ctx), 3)

	require.NoError(t, v0_21_0.MigrateGuildNameIndex(f.ctx, f.keepers()))

	index := f.k.GetAllGuildNameIndex(f.ctx)
	require.Equal(t, map[string]string{"alpha guild": kept.Id}, index,
		"only rows derived from a stored guild name may survive")

	_, taken := f.k.GetGuildIdByName(f.ctx, "Stale Name")
	require.False(t, taken, "a stale name must be free to claim again")
	_, taken = f.k.GetGuildIdByName(f.ctx, "Ghost Guild")
	require.False(t, taken)
}

// TestMigrateGuildNameIndex_DropsNameInvalidUnderPinnedRules exercises the
// state a divergent binary could have written: U+088F is unassigned in Unicode
// 15.0.0 and a letter in later versions, so a validator built with newer tables
// would have accepted a name made of it while every other validator rejected the
// same transaction. The name is dropped rather than carried forward, because an
// index row keyed to a name the chain would now refuse is worse than making the
// guild spend one transaction setting it again.
func TestMigrateGuildNameIndex_DropsNameInvalidUnderPinnedRules(t *testing.T) {
	f := newGuildNameFixture(t)

	postPinnedName := strings.Repeat("\u088F", 3)
	require.Error(t, types.ValidateEntityName(postPinnedName),
		"fixture assumption: this name must be invalid under the pinned tables")

	valid := f.named(0, "Alpha Guild")
	divergent := f.named(1, postPinnedName)
	tooShort := f.named(2, "ab")

	require.NoError(t, v0_21_0.MigrateGuildNameIndex(f.ctx, f.keepers()))

	require.Equal(t, map[string]string{"alpha guild": valid.Id}, f.k.GetAllGuildNameIndex(f.ctx),
		"only the valid name may keep a row")
	require.Equal(t, "", f.name(divergent.Id), "an unrepresentable name must be cleared, not kept")
	require.Equal(t, "", f.name(tooShort.Id))
	require.Equal(t, "Alpha Guild", f.name(valid.Id))

	// A cleared name is simply empty on a re-run, so the migration settles.
	require.NoError(t, v0_21_0.MigrateGuildNameIndex(f.ctx, f.keepers()))
	require.Equal(t, map[string]string{"alpha guild": valid.Id}, f.k.GetAllGuildNameIndex(f.ctx))
}

// TestMigrateGuildNameIndex_CollisionKeepsTheFirstGuild covers the case where
// two guilds want the same key. GuildUpdateName's uniqueness check makes this
// unreachable from state a correct binary wrote, so reaching it means the tables
// disagreed somewhere -- but the rebuild still has to resolve it identically on
// every node rather than halt the upgrade.
//
// GetAllGuild walks the guild prefix in key order, so the lower guild id wins
// everywhere.
func TestMigrateGuildNameIndex_CollisionKeepsTheFirstGuild(t *testing.T) {
	f := newGuildNameFixture(t)

	first := f.named(0, "Alpha Guild")
	second := f.named(1, "ALPHA GUILD")

	// Both are valid names that fold to one key, which is exactly what the
	// uniqueness check exists to prevent.
	require.NoError(t, types.ValidateEntityName(second.Name))
	require.Equal(t, types.NormalizeName(first.Name), types.NormalizeName(second.Name))

	require.NoError(t, v0_21_0.MigrateGuildNameIndex(f.ctx, f.keepers()))

	require.Equal(t, map[string]string{"alpha guild": first.Id}, f.k.GetAllGuildNameIndex(f.ctx),
		"the first guild in key order keeps the contested key")
	require.Equal(t, "Alpha Guild", f.name(first.Id))
	require.Equal(t, "", f.name(second.Id), "the losing guild's name must be dropped, not left unindexed")

	// With the loser's name cleared, a re-run has nothing left to contest.
	require.NoError(t, v0_21_0.MigrateGuildNameIndex(f.ctx, f.keepers()))
	require.Equal(t, map[string]string{"alpha guild": first.Id}, f.k.GetAllGuildNameIndex(f.ctx))
}

// TestMigrateGuildNameIndex_EmptyStateIsSafe guards the ordinary case for a
// chain with no guilds, since the migration runs unconditionally.
func TestMigrateGuildNameIndex_EmptyStateIsSafe(t *testing.T) {
	f := newGuildNameFixture(t)
	require.NoError(t, v0_21_0.MigrateGuildNameIndex(f.ctx, f.keepers()))
	require.Empty(t, f.k.GetAllGuildNameIndex(f.ctx))
}

// overdueFixture builds providers and agreements directly in the store, which is
// the shape the migration has to cope with: state left behind by an older binary,
// not state a healthy chain would produce.
type overdueFixture struct {
	k         structskeeper.Keeper
	ctx       sdk.Context
	denom     string
	providers map[string]types.Provider
}

func newOverdueFixture(t *testing.T, height int64) *overdueFixture {
	t.Helper()

	k, ctx := keepertest.StructsKeeper(t)

	return &overdueFixture{
		k:         k,
		ctx:       sdk.UnwrapSDKContext(ctx).WithBlockHeight(height),
		denom:     "ualpha",
		providers: map[string]types.Provider{},
	}
}

func (f *overdueFixture) keepers() *upgrades.Keepers {
	return &upgrades.Keepers{StructsKeeper: f.k}
}

// provider stores a provider with a published rate and seeds its agreement load
// grid attribute, so that a released agreement is visible as a decrement.
func (f *overdueFixture) provider(t *testing.T, id string, rate int64, load uint64) types.Provider {
	t.Helper()

	stored, err := f.k.SetProvider(f.ctx, types.Provider{
		Id:                          id,
		Rate:                        sdk.NewCoin(f.denom, math.NewInt(rate)),
		ProviderCancellationPenalty: math.LegacyMustNewDecFromStr("0"),
		ConsumerCancellationPenalty: math.LegacyMustNewDecFromStr("0"),
	})
	require.NoError(t, err)

	f.k.SetGridAttribute(f.ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_load, id), load)
	f.providers[id] = stored

	return stored
}

func (f *overdueFixture) agreement(t *testing.T, id string, providerId string, capacity uint64, startBlock uint64, endBlock uint64) types.Agreement {
	t.Helper()

	agreement := types.Agreement{
		Id:         id,
		ProviderId: providerId,
		Capacity:   capacity,
		StartBlock: startBlock,
		EndBlock:   endBlock,
	}
	_, err := f.k.SetAgreement(f.ctx, agreement)
	require.NoError(t, err)
	require.NoError(t, f.k.SetAgreementProviderIndex(f.ctx, providerId, id))
	require.NoError(t, f.k.SetAgreementExpirationIndex(f.ctx, endBlock, id))

	return agreement
}

func (f *overdueFixture) load(providerId string) uint64 {
	return f.k.GetGridAttribute(f.ctx, structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_load, providerId))
}

// TestMigrateExpireOverdueAgreements_SettlesTheStranded covers the state the
// migration exists for: an agreement whose end block has passed but which is
// still in the store, still holding its capacity in the provider's load.
//
// Expiry is driven from the EndBlocker at exactly the end block, with no range
// scan and no retry, so one that is missed is never revisited and the provider
// bills for it forever.
func TestMigrateExpireOverdueAgreements_SettlesTheStranded(t *testing.T) {
	const height = 5000
	f := newOverdueFixture(t, height)

	// 300 of load: 100 stranded past its end block, 200 still under way.
	provider := f.provider(t, "provider-stranded", 10, 300)
	stranded := f.agreement(t, "agreement-stranded", provider.Id, 100, 100, height-1)
	current := f.agreement(t, "agreement-current", provider.Id, 200, 100, height+500)

	require.NoError(t, v0_21_0.MigrateExpireOverdueAgreements(f.ctx, f.keepers()))

	_, found := f.k.GetAgreement(f.ctx, stranded.Id)
	require.False(t, found, "an agreement past its end block should have been settled")
	require.NotContains(t, f.k.GetAllAgreementIdByProviderIndex(f.ctx, provider.Id), stranded.Id)
	require.NotContains(t, f.k.GetAllAgreementIdByExpirationIndex(f.ctx, stranded.EndBlock), stranded.Id,
		"the expiration index row must go with it, or it is left pointing at nothing")

	_, found = f.k.GetAgreement(f.ctx, current.Id)
	require.True(t, found, "an agreement still inside its window must be left alone")

	require.Equal(t, uint64(200), f.load(provider.Id),
		"only the stranded agreement's capacity should have been released")

	// Which is the whole point: the invariant that would have flagged this is now
	// clean.
	msg, broken := structskeeper.AgreementExpiryLivenessInvariant(f.k)(f.ctx)
	require.False(t, broken, msg)
}

// TestMigrateExpireOverdueAgreements_HealthyStateIsUntouched is the expected case
// on a real chain: nothing is overdue, so nothing moves.
func TestMigrateExpireOverdueAgreements_HealthyStateIsUntouched(t *testing.T) {
	const height = 5000
	f := newOverdueFixture(t, height)

	provider := f.provider(t, "provider-healthy", 10, 150)

	// One mid-window, and one ending on this very block: the EndBlocker settles
	// that one itself, and the migration must not race it.
	open := f.agreement(t, "agreement-open", provider.Id, 100, 100, height+500)
	ending := f.agreement(t, "agreement-ending", provider.Id, 50, 100, height)

	require.NoError(t, v0_21_0.MigrateExpireOverdueAgreements(f.ctx, f.keepers()))

	for _, agreement := range []types.Agreement{open, ending} {
		stored, found := f.k.GetAgreement(f.ctx, agreement.Id)
		require.True(t, found, "agreement %s should be untouched", agreement.Id)
		require.Equal(t, agreement, stored)
	}

	require.Equal(t, uint64(150), f.load(provider.Id), "no load should have been released")
}

// TestMigrateExpireOverdueAgreements_EmptyStateIsSafe guards the walk itself.
func TestMigrateExpireOverdueAgreements_EmptyStateIsSafe(t *testing.T) {
	f := newOverdueFixture(t, 5000)
	require.NoError(t, v0_21_0.MigrateExpireOverdueAgreements(f.ctx, f.keepers()))
	require.Empty(t, f.k.GetAllAgreement(f.ctx))
}

// --- MigrateAutoResizeAllocationIndex ---------------------------------------
//
// The auto-resize index maps a source object id to the automated allocation
// riding on it. AllocationCache.Destroy cleared it with the allocation's own id,
// so the delete matched no key and every automated allocation ever torn down
// left its hook behind, naming an allocation that no longer exists. A leaked
// hook bricks its source: SetSource refuses a replacement on any source the
// index mentions, and the infusion capacity path treats the hook as live and so
// skips the grid cascade a capacity cut should trigger.
//
// The migration rebuilds the index from the allocations themselves, which fixes
// every way it can be wrong at once.

type autoResizeFixture struct {
	k   structskeeper.Keeper
	ctx sdk.Context
}

func newAutoResizeFixture(t *testing.T) *autoResizeFixture {
	t.Helper()
	k, ctx := keepertest.StructsKeeper(t)
	return &autoResizeFixture{k: k, ctx: ctx}
}

func (f *autoResizeFixture) keepers() *upgrades.Keepers {
	return &upgrades.Keepers{StructsKeeper: f.k}
}

// allocation writes an allocation record without touching any index, so each
// test states the index contents explicitly rather than inheriting them.
func (f *autoResizeFixture) allocation(id string, sourceId string, allocationType types.AllocationType) types.Allocation {
	allocation := types.Allocation{
		Id:             id,
		SourceObjectId: sourceId,
		Type:           allocationType,
	}
	f.k.ImportAllocation(f.ctx, allocation)
	return allocation
}

func (f *autoResizeFixture) hook(sourceId string) (string, bool) {
	return f.k.GetAutoResizeAllocationBySource(f.ctx, sourceId)
}

// index returns the whole index as a source-to-allocation map.
func (f *autoResizeFixture) index() map[string]string {
	out := make(map[string]string)
	for _, hook := range f.k.GetAllAutoResizeAllocationSource(f.ctx) {
		out[hook.SourceObjectId] = hook.AllocationId
	}
	return out
}

// TestMigrateAutoResizeAllocationIndex_DropsStaleHooks is the case the leak
// actually produces: the allocation is gone, the hook is not.
func TestMigrateAutoResizeAllocationIndex_DropsStaleHooks(t *testing.T) {
	f := newAutoResizeFixture(t)

	// A live automated allocation that must survive the rebuild.
	live := f.allocation("6-0", "1-0", types.AllocationType_automated)
	f.k.SetAutoResizeAllocationSource(f.ctx, live.Id, live.SourceObjectId)

	// A hook whose allocation was destroyed. Nothing wrote allocation 6-99.
	f.k.SetAutoResizeAllocationSource(f.ctx, "6-99", "1-1")

	require.NoError(t, v0_21_0.MigrateAutoResizeAllocationIndex(f.ctx, f.keepers()))

	require.Equal(t, map[string]string{"1-0": live.Id}, f.index(),
		"only the hook naming a live automated allocation should remain")

	_, found := f.hook("1-1")
	require.False(t, found, "the source of a destroyed automated allocation must be usable again")
}

// TestMigrateAutoResizeAllocationIndex_DropsNonAutomatedAndRekeysWrongSource
// covers the two inconsistencies a rebuild catches that a prune would not.
func TestMigrateAutoResizeAllocationIndex_DropsNonAutomatedAndRekeysWrongSource(t *testing.T) {
	f := newAutoResizeFixture(t)

	// A hook naming an allocation that exists but is not automated.
	dynamic := f.allocation("6-0", "1-0", types.AllocationType_dynamic)
	f.k.SetAutoResizeAllocationSource(f.ctx, dynamic.Id, dynamic.SourceObjectId)

	// A hook filed under a source its allocation does not claim.
	automated := f.allocation("6-1", "1-1", types.AllocationType_automated)
	f.k.SetAutoResizeAllocationSource(f.ctx, automated.Id, "1-2")

	require.NoError(t, v0_21_0.MigrateAutoResizeAllocationIndex(f.ctx, f.keepers()))

	require.Equal(t, map[string]string{"1-1": automated.Id}, f.index(),
		"a non-automated allocation should not be hooked, and a hook belongs under the source its allocation names")
}

// TestMigrateAutoResizeAllocationIndex_HealthyStateIsUnchanged is the outcome on
// a chain that never destroyed an automated allocation: the rebuild replaces the
// index with itself.
func TestMigrateAutoResizeAllocationIndex_HealthyStateIsUnchanged(t *testing.T) {
	f := newAutoResizeFixture(t)

	first := f.allocation("6-0", "1-0", types.AllocationType_automated)
	second := f.allocation("6-1", "1-1", types.AllocationType_automated)
	f.allocation("6-2", "1-2", types.AllocationType_dynamic)

	f.k.SetAutoResizeAllocationSource(f.ctx, first.Id, first.SourceObjectId)
	f.k.SetAutoResizeAllocationSource(f.ctx, second.Id, second.SourceObjectId)

	before := f.index()
	require.NoError(t, v0_21_0.MigrateAutoResizeAllocationIndex(f.ctx, f.keepers()))
	require.Equal(t, before, f.index())
}

// TestMigrateAutoResizeAllocationIndex_CollisionKeepsTheFirst pins the tie-break
// for corrupt state, which must not depend on Go map ordering.
func TestMigrateAutoResizeAllocationIndex_CollisionKeepsTheFirst(t *testing.T) {
	f := newAutoResizeFixture(t)

	f.allocation("6-0", "1-0", types.AllocationType_automated)
	f.allocation("6-1", "1-0", types.AllocationType_automated)

	require.NoError(t, v0_21_0.MigrateAutoResizeAllocationIndex(f.ctx, f.keepers()))

	require.Equal(t, map[string]string{"1-0": "6-0"}, f.index(),
		"the first allocation in store key order must win, on every node")
}

// TestMigrateAutoResizeAllocationIndex_IsIdempotent guards a replay of the
// upgrade block.
func TestMigrateAutoResizeAllocationIndex_IsIdempotent(t *testing.T) {
	f := newAutoResizeFixture(t)

	live := f.allocation("6-0", "1-0", types.AllocationType_automated)
	f.k.SetAutoResizeAllocationSource(f.ctx, live.Id, live.SourceObjectId)
	f.k.SetAutoResizeAllocationSource(f.ctx, "6-99", "1-1")

	require.NoError(t, v0_21_0.MigrateAutoResizeAllocationIndex(f.ctx, f.keepers()))
	once := f.index()

	require.NoError(t, v0_21_0.MigrateAutoResizeAllocationIndex(f.ctx, f.keepers()))
	require.Equal(t, once, f.index())
}

// TestMigrateAutoResizeAllocationIndex_EmptyStateIsSafe guards the walk itself.
func TestMigrateAutoResizeAllocationIndex_EmptyStateIsSafe(t *testing.T) {
	f := newAutoResizeFixture(t)
	require.NoError(t, v0_21_0.MigrateAutoResizeAllocationIndex(f.ctx, f.keepers()))
	require.Empty(t, f.index())
}

// --- MigrateOrphanedAllocationControllers -----------------------------------
//
// AllocationTransfer checked msg.Controller with a guard on
// CurrentContext.GetPlayer, which cannot fail, so the id went unchecked and a
// transfer to any string committed. The allocation ends up controlled by somebody
// who can never sign for it, with a permission row keyed to them.
//
// The repair hands such an allocation to the owner of its source, with the same
// single bit a legitimate transfer would have granted.

type orphanFixture struct {
	t   *testing.T
	k   structskeeper.Keeper
	ctx sdk.Context
}

func newOrphanFixture(t *testing.T) *orphanFixture {
	t.Helper()
	k, ctx := keepertest.StructsKeeper(t)
	return &orphanFixture{t: t, k: k, ctx: ctx}
}

func (f *orphanFixture) keepers() *upgrades.Keepers {
	return &upgrades.Keepers{StructsKeeper: f.k}
}

// player puts a real player in state. A player is its own owner, so using one as
// an allocation's source object makes the expected new controller that player.
func (f *orphanFixture) player(id string) types.Player {
	player := types.Player{Id: id}
	f.k.SetPlayer(f.ctx, player)
	return player
}

func (f *orphanFixture) allocation(id string, sourceId string, controller string) types.Allocation {
	allocation := types.Allocation{
		Id:             id,
		SourceObjectId: sourceId,
		Type:           types.AllocationType_static,
		Controller:     controller,
	}
	f.k.ImportAllocation(f.ctx, allocation)
	return allocation
}

func (f *orphanFixture) controllerOf(id string) string {
	allocation, found := f.k.GetAllocation(f.ctx, id)
	require.True(f.t, found)
	return allocation.Controller
}

func (f *orphanFixture) perms(allocationId string, playerId string) types.Permission {
	return f.k.GetPermissionsByBytes(f.ctx, structskeeper.GetObjectPermissionIDBytes(allocationId, playerId))
}

// TestMigrateOrphanedAllocationControllers_RehomesToSourceOwner is the case the
// bug produces, and pins the permissions the repair is allowed to touch.
func TestMigrateOrphanedAllocationControllers_RehomesToSourceOwner(t *testing.T) {
	f := newOrphanFixture(t)

	owner := f.player("1-0")
	orphan := f.allocation("6-0", owner.Id, "1-999")

	// What the bad transfer left behind, plus the creator's own row on the same
	// allocation, which the repair must not disturb.
	f.k.SetPermissionsByBytes(f.ctx, structskeeper.GetObjectPermissionIDBytes(orphan.Id, "1-999"),
		types.PermAllocationConnection)
	f.k.SetPermissionsByBytes(f.ctx, structskeeper.GetObjectPermissionIDBytes(orphan.Id, owner.Id),
		types.PermUpdate|types.PermDelete)

	require.NoError(t, v0_21_0.MigrateOrphanedAllocationControllers(f.ctx, f.keepers()))

	require.Equal(t, owner.Id, f.controllerOf(orphan.Id),
		"an allocation controlled by nobody belongs to the owner of its source")

	require.Equal(t, types.Permissionless, f.perms(orphan.Id, "1-999"),
		"the row keyed to a player that does not exist should be gone")

	ownerPerms := f.perms(orphan.Id, owner.Id)
	require.NotZero(t, ownerPerms&types.PermAllocationConnection,
		"the new controller can connect what they now control")
	require.Zero(t, ownerPerms&types.PermAdmin,
		"the repair grants exactly what a transfer grants, and a transfer does not mint PermAdmin")
	require.NotZero(t, ownerPerms&types.PermDelete,
		"bits the owner already held are left alone")
	require.NotZero(t, ownerPerms&types.PermUpdate,
		"bits the owner already held are left alone")
}

// TestMigrateOrphanedAllocationControllers_LeavesAllocationWithNoUsableOwner
// covers the case where there is nobody to hand it to. Guessing would be worse
// than leaving it: the source owner can still reach it through
// PermSourceAllocation on the source.
func TestMigrateOrphanedAllocationControllers_LeavesAllocationWithNoUsableOwner(t *testing.T) {
	f := newOrphanFixture(t)

	// The source names a player who is not in state either, so GetOwnerId reports
	// an id that resolves to nobody.
	orphan := f.allocation("6-0", "1-404", "1-999")

	require.NoError(t, v0_21_0.MigrateOrphanedAllocationControllers(f.ctx, f.keepers()))

	require.Equal(t, "1-999", f.controllerOf(orphan.Id),
		"with no owner to hand it to the allocation is left as found")
}

// TestMigrateOrphanedAllocationControllers_HealthyStateIsUnchanged is the outcome
// on a chain where no bad transfer ever happened, which is the expected one.
func TestMigrateOrphanedAllocationControllers_HealthyStateIsUnchanged(t *testing.T) {
	f := newOrphanFixture(t)

	source := f.player("1-0")
	controller := f.player("1-1")
	healthy := f.allocation("6-0", source.Id, controller.Id)
	f.k.SetPermissionsByBytes(f.ctx, structskeeper.GetObjectPermissionIDBytes(healthy.Id, controller.Id),
		types.PermAllocationConnection|types.PermAdmin)

	require.NoError(t, v0_21_0.MigrateOrphanedAllocationControllers(f.ctx, f.keepers()))

	require.Equal(t, controller.Id, f.controllerOf(healthy.Id))
	require.Equal(t, types.PermAllocationConnection|types.PermAdmin, f.perms(healthy.Id, controller.Id),
		"a healthy allocation keeps the PermAdmin a real transfer left it")
	require.Equal(t, types.Permissionless, f.perms(healthy.Id, source.Id))
}

// TestMigrateOrphanedAllocationControllers_IsIdempotent guards a replay of the
// upgrade block.
func TestMigrateOrphanedAllocationControllers_IsIdempotent(t *testing.T) {
	f := newOrphanFixture(t)

	owner := f.player("1-0")
	orphan := f.allocation("6-0", owner.Id, "1-999")

	require.NoError(t, v0_21_0.MigrateOrphanedAllocationControllers(f.ctx, f.keepers()))
	once := f.controllerOf(orphan.Id)
	oncePerms := f.perms(orphan.Id, owner.Id)

	require.NoError(t, v0_21_0.MigrateOrphanedAllocationControllers(f.ctx, f.keepers()))
	require.Equal(t, once, f.controllerOf(orphan.Id))
	require.Equal(t, oncePerms, f.perms(orphan.Id, owner.Id))
}

// TestMigrateOrphanedAllocationControllers_EmptyStateIsSafe guards the walk.
func TestMigrateOrphanedAllocationControllers_EmptyStateIsSafe(t *testing.T) {
	f := newOrphanFixture(t)
	require.NoError(t, v0_21_0.MigrateOrphanedAllocationControllers(f.ctx, f.keepers()))
}

// reconcileFixture builds reactor infusions through the real
// AfterDelegationModified path, so their grid attributes carry the values a live
// chain would have rather than hand-seeded ones. A phantom is then made the way
// the chain made them: by taking the delegation away and leaving the infusion.
type reconcileFixture struct {
	t    *testing.T
	k    structskeeper.Keeper
	ctx  sdk.Context
	mock *keepertest.MockStakingKeeper
}

func newReconcileFixture(t *testing.T) *reconcileFixture {
	t.Helper()

	k, ctx := keepertest.StructsKeeper(t)
	return &reconcileFixture{
		t:    t,
		k:    k,
		ctx:  ctx,
		mock: k.StakingKeeper().(*keepertest.MockStakingKeeper),
	}
}

func (f *reconcileFixture) keepers() *upgrades.Keepers {
	return &upgrades.Keepers{StructsKeeper: f.k}
}

func (f *reconcileFixture) addReactor(seed string, tokens int64) (types.Reactor, sdk.ValAddress) {
	f.t.Helper()

	valAddr := sdk.ValAddress(fmt.Sprintf("%-36s", seed)[:36])
	f.mock.AddValidator(valAddr, math.NewInt(tokens))

	reactor := f.k.AppendReactor(f.ctx, types.Reactor{
		Validator:         valAddr.String(),
		RawAddress:        valAddr.Bytes(),
		DefaultCommission: math.LegacyMustNewDecFromStr("0.04"),
	})

	return reactor, valAddr
}

func (f *reconcileFixture) addPlayer(seed string) (types.Player, sdk.AccAddress) {
	f.t.Helper()

	playerAcc := sdk.AccAddress(fmt.Sprintf("%-36s", seed)[:36])
	player := types.Player{Creator: playerAcc.String(), PrimaryAddress: playerAcc.String()}
	player.Index = f.k.GetPlayerCount(f.ctx)
	player.Id = fmt.Sprintf("%d-%d", types.ObjectType_player, player.Index)
	f.k.SetPlayer(f.ctx, player)
	f.k.SetPlayerCount(f.ctx, player.Index+1)
	f.k.SetPlayerIndexForAddress(f.ctx, player.PrimaryAddress, player.Index)

	return player, playerAcc
}

func (f *reconcileFixture) infuse(playerAcc sdk.AccAddress, valAddr sdk.ValAddress, tokens int64) {
	f.t.Helper()

	require.NoError(f.t, f.mock.SetDelegation(f.ctx, stakingtypes.Delegation{
		DelegatorAddress: playerAcc.String(),
		ValidatorAddress: valAddr.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(tokens)),
	}))

	f.k.ReactorUpdatePlayerInfusion(f.ctx, playerAcc, valAddr)
}

// dropDelegation reproduces the corruption: staking loses the delegation while
// the infusion behind it stays exactly as it was. This is what a full
// redelegation left on chain while BeforeDelegationRemoved was a no-op.
func (f *reconcileFixture) dropDelegation(playerAcc sdk.AccAddress, valAddr sdk.ValAddress) {
	f.t.Helper()

	require.NoError(f.t, f.mock.RemoveDelegation(f.ctx, stakingtypes.Delegation{
		DelegatorAddress: playerAcc.String(),
		ValidatorAddress: valAddr.String(),
	}))
}

func (f *reconcileFixture) capacity(objectId string) uint64 {
	return f.k.GetGridAttribute(f.ctx,
		structskeeper.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId))
}

func (f *reconcileFixture) infusion(reactorId string, playerAcc sdk.AccAddress) types.Infusion {
	f.t.Helper()

	infusion, found := f.k.GetInfusion(f.ctx, reactorId, playerAcc.String())
	require.True(f.t, found, "infusion should exist")
	return infusion
}

// TestMigrateReconcileReactorInfusions_ClearsPhantom is the state repair for the
// redelegation leak. One player holds infusions at two reactors but has only one
// delegation, which is precisely the shape a full redelegation left behind: the
// destination is real, the source is capacity backed by nothing.
func TestMigrateReconcileReactorInfusions_ClearsPhantom(t *testing.T) {
	f := newReconcileFixture(t)

	sourceReactor, sourceVal := f.addReactor("reconcilesource", 1000)
	destReactor, destVal := f.addReactor("reconciledest", 1000)
	player, playerAcc := f.addPlayer("reconcileplayer")

	f.infuse(playerAcc, sourceVal, 1000)
	f.infuse(playerAcc, destVal, 1000)

	// At 4% commission each infusion gives the player 960 and its reactor 40.
	require.Equal(t, uint64(1920), f.capacity(player.Id), "two infusions, two player shares")
	require.Equal(t, uint64(40), f.capacity(sourceReactor.Id))

	f.dropDelegation(playerAcc, sourceVal)

	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))

	source := f.infusion(sourceReactor.Id, playerAcc)
	require.Equal(t, uint64(0), source.Fuel, "fuel with no delegation behind it must go")
	require.Equal(t, uint64(0), source.Power)
	require.Equal(t, uint64(0), f.capacity(sourceReactor.Id), "the source reactor loses its commission share")

	dest := f.infusion(destReactor.Id, playerAcc)
	require.Equal(t, uint64(1000), dest.Fuel, "the real delegation is untouched")
	require.Equal(t, uint64(1000), dest.Power)
	require.Equal(t, uint64(40), f.capacity(destReactor.Id))

	require.Equal(t, uint64(960), f.capacity(player.Id),
		"the player keeps exactly the capacity one stake buys")
}

// TestMigrateReconcileReactorInfusions_LeavesHealthyStateAlone guards the blast
// radius from the other side. The migration walks every infusion on the chain,
// so an unguarded write would re-emit EventInfusion for every healthy row and
// flood every downstream indexer.
func TestMigrateReconcileReactorInfusions_LeavesHealthyStateAlone(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("reconcilehealthy", 1000)
	player, playerAcc := f.addPlayer("reconcilehealthyplayer")
	f.infuse(playerAcc, valAddr, 1000)

	before := f.infusion(reactor.Id, playerAcc)
	eventsBefore := len(f.ctx.EventManager().Events())

	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))

	require.Equal(t, before, f.infusion(reactor.Id, playerAcc), "a healthy row must be left byte-identical")
	require.Equal(t, uint64(960), f.capacity(player.Id))
	require.Equal(t, eventsBefore, len(f.ctx.EventManager().Events()),
		"reconciling healthy state must emit nothing")
}

// TestMigrateReconcileReactorInfusions_PreservesDefusing covers a player who was
// mid-withdrawal when the upgrade lands: they unbonded part of their stake and
// moved the rest. The unbonding balance is owed to them regardless of the
// phantom fuel sitting next to it.
//
// The balance has to be a real unbonding delegation, because Defusing is
// derived from staking rather than trusted from the record. That is deliberate,
// and it is the half of the reconcile that repairs a stored value rather than
// preserving it.
func TestMigrateReconcileReactorInfusions_PreservesDefusing(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("reconciledefusing", 1000)
	_, playerAcc := f.addPlayer("reconciledefusingplayer")
	f.infuse(playerAcc, valAddr, 1000)

	require.NoError(t, f.mock.SetUnbondingDelegation(f.ctx, stakingtypes.UnbondingDelegation{
		DelegatorAddress: playerAcc.String(),
		ValidatorAddress: valAddr.String(),
		Entries: []stakingtypes.UnbondingDelegationEntry{
			{Balance: math.NewInt(250), CompletionTime: time.Now().UTC().Add(time.Hour)},
		},
	}))

	f.dropDelegation(playerAcc, valAddr)

	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))

	after := f.infusion(reactor.Id, playerAcc)
	require.Equal(t, uint64(0), after.Fuel, "the phantom fuel still goes")
	require.Equal(t, uint64(0), after.Power)
	require.Equal(t, uint64(250), after.Defusing, "an unbonding balance is not phantom capacity")
}

// TestMigrateReconcileReactorInfusions_IsIdempotent guards a replayed upgrade
// block.
func TestMigrateReconcileReactorInfusions_IsIdempotent(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("reconcilereplay", 1000)
	player, playerAcc := f.addPlayer("reconcilereplayplayer")
	f.infuse(playerAcc, valAddr, 1000)
	f.dropDelegation(playerAcc, valAddr)

	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))
	once := f.infusion(reactor.Id, playerAcc)
	onceCapacity := f.capacity(player.Id)
	eventsAfterFirst := len(f.ctx.EventManager().Events())

	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))

	require.Equal(t, once, f.infusion(reactor.Id, playerAcc))
	require.Equal(t, onceCapacity, f.capacity(player.Id))
	require.Equal(t, eventsAfterFirst, len(f.ctx.EventManager().Events()),
		"a replay must write nothing the first pass did not")
}

// TestMigrateReconcileReactorInfusions_HealsUnregisteredAddress covers the
// phantom capacity a pre-fix AddressRevoke left behind: the infused address was
// unregistered while its infusion kept full fuel/power. The migration must heal
// it (not skip it), zeroing the fuel behind stake that no longer backs it,
// without minting a player for the dead address.
func TestMigrateReconcileReactorInfusions_HealsUnregisteredAddress(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("reconcileunreg", 1000)
	player, playerAcc := f.addPlayer("reconcileunregplayer")
	f.infuse(playerAcc, valAddr, 1000)

	require.Equal(t, uint64(960), f.capacity(player.Id), "one infusion, one player share")

	// Reproduce a pre-fix revoke: the stake left the address (no delegation) and
	// the address was unregistered, but the infusion record still carries fuel.
	f.dropDelegation(playerAcc, valAddr)
	f.k.RevokePlayerIndexForAddress(f.ctx, playerAcc.String(), player.Index)
	require.Equal(t, uint64(0), f.k.GetPlayerIndexFromAddress(f.ctx, playerAcc.String()),
		"address is unregistered going into the migration")

	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))

	healed := f.infusion(reactor.Id, playerAcc)
	require.Equal(t, uint64(0), healed.Fuel, "phantom fuel behind an unregistered address must go")
	require.Equal(t, uint64(0), healed.Power)
	require.Equal(t, uint64(0), f.capacity(player.Id), "the player loses capacity no stake backs")
	require.Equal(t, uint64(0), f.k.GetPlayerIndexFromAddress(f.ctx, playerAcc.String()),
		"the migration must not register a player for the dead address")
}

// TestMigrateReconcileReactorInfusions_EmptyStateIsSafe guards the walk itself.
func TestMigrateReconcileReactorInfusions_EmptyStateIsSafe(t *testing.T) {
	f := newReconcileFixture(t)
	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))
}

// TestMigrateReconcileReactorInfusions_SkipsUnparsableRows keeps the walk going
// past a record it cannot resolve, so one malformed row cannot abort the upgrade
// and strand every phantom behind it.
func TestMigrateReconcileReactorInfusions_SkipsUnparsableRows(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("reconcileskip", 1000)
	_, playerAcc := f.addPlayer("reconcileskipplayer")
	f.infuse(playerAcc, valAddr, 1000)
	f.dropDelegation(playerAcc, valAddr)

	f.k.SetInfusion(f.ctx, types.Infusion{
		DestinationType: types.ObjectType_reactor,
		DestinationId:   reactor.Id,
		Address:         "not-a-bech32-address",
		PlayerId:        "1-999",
		Commission:      math.LegacyZeroDec(),
		Fuel:            500,
	})

	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))

	require.Equal(t, uint64(0), f.infusion(reactor.Id, playerAcc).Fuel,
		"the resolvable phantom is still cleared")
}

func TestMigrateReconcileReactorInfusions_DoesNotRecreateRevokedAddress(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("reconcilerevoked", 1000)
	player, playerAcc := f.addPlayer("reconcilerevokedplayer")
	f.infuse(playerAcc, valAddr, 1000)

	f.k.RevokePlayerIndexForAddress(f.ctx, playerAcc.String(), player.Index)
	playerCount := f.k.GetPlayerCount(f.ctx)
	before := f.infusion(reactor.Id, playerAcc)

	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))

	require.Equal(t, uint64(0), f.k.GetPlayerIndexFromAddress(f.ctx, playerAcc.String()))
	require.Equal(t, playerCount, f.k.GetPlayerCount(f.ctx),
		"reconcile must not mint a replacement player for a revoked key")
	require.Equal(t, before, f.infusion(reactor.Id, playerAcc),
		"an unregistered address is intentionally left for a future registration to re-home")
}

// reassignAddress hands an address from one player to the next, which is what
// AddressRevoke followed by AddressRegister leaves in the index and what
// UpsertInfusion used to ignore.
func (f *reconcileFixture) reassignAddress(playerAcc sdk.AccAddress, from types.Player, to types.Player) {
	f.t.Helper()

	f.k.RevokePlayerIndexForAddress(f.ctx, playerAcc.String(), from.Index)
	require.NoError(f.t, f.k.SetPlayerIndexForAddress(f.ctx, playerAcc.String(), to.Index))
}

// TestMigrateInfusionOwnership_RehomesStaleRow is the state repair for the
// ownership leak. The address changed hands and the record did not, so the
// former owner kept both the capacity and, through GuildMembershipJoin, the
// authority to redelegate stake they no longer control.
func TestMigrateInfusionOwnership_RehomesStaleRow(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("ownershipstale", 1000)
	formerOwner, playerAcc := f.addPlayer("ownershipformer")
	newOwner, _ := f.addPlayer("ownershipnew")

	f.infuse(playerAcc, valAddr, 1000)
	require.Equal(t, formerOwner.Id, f.infusion(reactor.Id, playerAcc).PlayerId)
	require.Equal(t, uint64(960), f.capacity(formerOwner.Id))

	f.reassignAddress(playerAcc, formerOwner, newOwner)

	require.NoError(t, v0_21_0.MigrateInfusionOwnership(f.ctx, f.keepers()))

	after := f.infusion(reactor.Id, playerAcc)
	require.Equal(t, newOwner.Id, after.PlayerId, "the record must follow the address")
	require.Equal(t, uint64(0), f.capacity(formerOwner.Id),
		"the former owner stops being paid for stake they cannot reach")
	require.Equal(t, uint64(960), f.capacity(newOwner.Id))

	require.Equal(t, uint64(1000), after.Fuel, "a re-home moves nobody's stake")
	require.Equal(t, uint64(1000), after.Power)
	require.Equal(t, uint64(40), f.capacity(reactor.Id),
		"the reactor's commission is keyed by destination and is not the delegator's to move")

	// Idempotent: a replayed upgrade block must not double-credit the new owner.
	eventsAfterFirst := len(f.ctx.EventManager().Events())
	require.NoError(t, v0_21_0.MigrateInfusionOwnership(f.ctx, f.keepers()))

	require.Equal(t, after, f.infusion(reactor.Id, playerAcc))
	require.Equal(t, uint64(960), f.capacity(newOwner.Id))
	require.Equal(t, eventsAfterFirst, len(f.ctx.EventManager().Events()),
		"a replay must write nothing the first pass did not")
}

// TestMigrateInfusionOwnership_LeavesCorrectRowsAlone guards the blast radius.
// The migration walks every infusion on the chain, and the overwhelming majority
// are correct, so an unguarded write would re-emit EventInfusion for all of them.
func TestMigrateInfusionOwnership_LeavesCorrectRowsAlone(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("ownershiphealthy", 1000)
	player, playerAcc := f.addPlayer("ownershiphealthyplayer")
	f.infuse(playerAcc, valAddr, 1000)

	before := f.infusion(reactor.Id, playerAcc)
	eventsBefore := len(f.ctx.EventManager().Events())

	require.NoError(t, v0_21_0.MigrateInfusionOwnership(f.ctx, f.keepers()))

	require.Equal(t, before, f.infusion(reactor.Id, playerAcc), "a correct row must be left byte-identical")
	require.Equal(t, uint64(960), f.capacity(player.Id))
	require.Equal(t, eventsBefore, len(f.ctx.EventManager().Events()),
		"re-homing nothing must emit nothing")
}

// TestMigrateInfusionOwnership_SkipsUnregisteredAddress covers an address that
// was revoked and never re-registered. There is nobody to re-home to, and
// stripping the capacity would punish whoever still holds the stake for a bug
// that was never theirs.
func TestMigrateInfusionOwnership_SkipsUnregisteredAddress(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("ownershiporphan", 1000)
	player, playerAcc := f.addPlayer("ownershiporphanplayer")
	f.infuse(playerAcc, valAddr, 1000)

	f.k.RevokePlayerIndexForAddress(f.ctx, playerAcc.String(), player.Index)

	require.NoError(t, v0_21_0.MigrateInfusionOwnership(f.ctx, f.keepers()))

	require.Equal(t, player.Id, f.infusion(reactor.Id, playerAcc).PlayerId,
		"an unowned address leaves ownership where it was")
	require.Equal(t, uint64(960), f.capacity(player.Id))
}

// TestMigrateInfusionOwnership_EmptyStateIsSafe guards the walk itself.
func TestMigrateInfusionOwnership_EmptyStateIsSafe(t *testing.T) {
	f := newReconcileFixture(t)
	require.NoError(t, v0_21_0.MigrateInfusionOwnership(f.ctx, f.keepers()))
}

// TestMigrateInfusionOwnership_RunsBeforeReconcile pins the registration order.
// The reconcile recomputes each row's fuel and therefore the capacity credited
// to its owner, so running it first would rebuild the same misattribution this
// migration exists to correct.
func TestMigrateInfusionOwnership_RunsBeforeReconcile(t *testing.T) {
	f := newReconcileFixture(t)

	reactor, valAddr := f.addReactor("ownershiporder", 1000)
	formerOwner, playerAcc := f.addPlayer("ownershiporderformer")
	newOwner, _ := f.addPlayer("ownershipordernew")

	f.infuse(playerAcc, valAddr, 1000)
	f.reassignAddress(playerAcc, formerOwner, newOwner)

	// The order the upgrade handler registers them in.
	require.NoError(t, v0_21_0.MigrateInfusionOwnership(f.ctx, f.keepers()))
	require.NoError(t, v0_21_0.MigrateReconcileReactorInfusions(f.ctx, f.keepers()))

	require.Equal(t, newOwner.Id, f.infusion(reactor.Id, playerAcc).PlayerId)
	require.Equal(t, uint64(0), f.capacity(formerOwner.Id))
	require.Equal(t, uint64(960), f.capacity(newOwner.Id),
		"the reconcile must credit the corrected owner, not rebuild the old one")
	require.Equal(t, uint64(1000), f.infusion(reactor.Id, playerAcc).Fuel,
		"the live delegation is left where it is")
}
