package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	keeperlib "structs/x/structs/keeper"
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

func testAppendPlayer(k keeperlib.Keeper, ctx context.Context, player types.Player) types.Player {
	count := k.GetPlayerCount(ctx)
	player.Index = count
	player.Id = fmt.Sprintf("%d-%d", types.ObjectType_player, count)
	k.SetPlayer(ctx, player)
	k.SetPlayerCount(ctx, count+1)
	if player.PrimaryAddress != "" {
		k.SetPlayerIndexForAddress(ctx, player.PrimaryAddress, player.Index)
	}
	addressPermId := keeperlib.GetAddressPermissionIDBytes(player.PrimaryAddress)
	k.SetPermissionsByBytes(ctx, addressPermId, types.PermAll)
	return player
}

func testAppendAllocation(k keeperlib.Keeper, ctx context.Context, allocation types.Allocation, power uint64) (types.Allocation, error) {
	count := k.GetAllocationCount(ctx)
	allocation.Index = count
	allocation.Id = fmt.Sprintf("%d-%d", types.ObjectType_allocation, count)
	k.SetAllocationCount(ctx, count+1)

	k.ImportAllocation(ctx, allocation)

	if power > 0 {
		powerAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, allocation.Id)
		k.SetGridAttribute(ctx, powerAttrId, power)
	}

	return allocation, nil
}

func testAppendSubstation(k keeperlib.Keeper, ctx context.Context, allocation types.Allocation, owner types.Player) (types.Substation, uint64, error) {
	count := k.GetSubstationCount(ctx)
	substation := types.Substation{
		Id:      fmt.Sprintf("%d-%d", types.ObjectType_substation, count),
		Owner:   owner.Id,
		Creator: owner.Creator,
	}
	k.SetSubstationCount(ctx, count+1)
	k.SetSubstation(ctx, substation)
	return substation, count, nil
}

func testAppendProvider(k keeperlib.Keeper, ctx context.Context, provider types.Provider) types.Provider {
	count := k.GetProviderCount(ctx)
	provider.Index = count
	provider.Id = fmt.Sprintf("%d-%d", types.ObjectType_provider, count)
	k.SetProviderCount(ctx, count+1)
	k.ImportProvider(ctx, provider)
	return provider
}

func testAppendStruct(k keeperlib.Keeper, ctx context.Context, structure types.Struct) types.Struct {
	count := k.GetStructCount(ctx)
	structure.Index = count
	structure.Id = fmt.Sprintf("%d-%d", types.ObjectType_struct, count)
	k.SetStructCount(ctx, count+1)
	k.SetStruct(ctx, structure)
	return structure
}

func testAppendPlanet(k keeperlib.Keeper, ctx context.Context, planet types.Planet) types.Planet {
	count := k.GetPlanetCount(ctx)
	planet.Id = fmt.Sprintf("%d-%d", types.ObjectType_planet, count)
	if planet.Status == 0 {
		planet.Status = types.PlanetStatus_active
	}
	k.SetPlanetCount(ctx, count+1)
	k.SetPlanet(ctx, planet)

	shieldAttrId := keeperlib.GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, planet.Id)
	k.SetPlanetAttribute(ctx, shieldAttrId, types.PlanetaryShieldBase)

	oreAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, planet.Id)
	k.SetGridAttribute(ctx, oreAttrId, types.PlanetStartingOre)

	return planet
}

func testAppendFleet(k keeperlib.Keeper, ctx context.Context, fleet types.Fleet) types.Fleet {
	ownerParts := strings.Split(fleet.Owner, "-")
	index := "0"
	if len(ownerParts) == 2 {
		index = ownerParts[1]
	}
	fleet.Id = fmt.Sprintf("%d-%s", types.ObjectType_fleet, index)
	k.SetFleet(ctx, fleet)
	return fleet
}

func testAppendReactor(k keeperlib.Keeper, ctx context.Context, reactor types.Reactor) types.Reactor {
	if reactor.RawAddress == nil {
		reactor.RawAddress = []byte("test-validator")
	}
	return k.AppendReactor(ctx, reactor)
}

func testAppendInfusion(k keeperlib.Keeper, ctx context.Context, infusion types.Infusion) types.Infusion {
	k.SetInfusion(ctx, infusion)
	return infusion
}

func testPermissionAdd(k keeperlib.Keeper, ctx context.Context, permissionId []byte, perm types.Permission) {
	existing := k.GetPermissionsByBytes(ctx, permissionId)
	k.SetPermissionsByBytes(ctx, permissionId, existing|perm)
}

func testPermissionRemove(k keeperlib.Keeper, ctx context.Context, permissionId []byte, perm types.Permission) {
	existing := k.GetPermissionsByBytes(ctx, permissionId)
	k.SetPermissionsByBytes(ctx, permissionId, existing&^perm)
}

func testPermissionHasAll(k keeperlib.Keeper, ctx context.Context, permissionId []byte, perm types.Permission) bool {
	existing := k.GetPermissionsByBytes(ctx, permissionId)
	return existing&perm == perm
}

func testPermissionHasOneOf(k keeperlib.Keeper, ctx context.Context, permissionId []byte, perm types.Permission) bool {
	existing := k.GetPermissionsByBytes(ctx, permissionId)
	return existing&perm != 0
}

func testSetStructAttributeFlagAdd(k keeperlib.Keeper, ctx context.Context, structAttributeId string, flag uint64) {
	current := k.GetStructAttribute(ctx, structAttributeId)
	k.SetStructAttribute(ctx, structAttributeId, current|flag)
}

func testSetStructAttributeFlagRemove(k keeperlib.Keeper, ctx context.Context, structAttributeId string, flag uint64) {
	current := k.GetStructAttribute(ctx, structAttributeId)
	k.SetStructAttribute(ctx, structAttributeId, current&^flag)
}

func testStructAttributeFlagHasAll(k keeperlib.Keeper, ctx context.Context, structAttributeId string, flag uint64) bool {
	return k.GetStructAttribute(ctx, structAttributeId)&flag == flag
}

func testStructAttributeFlagHasOneOf(k keeperlib.Keeper, ctx context.Context, structAttributeId string, flag uint64) bool {
	return k.GetStructAttribute(ctx, structAttributeId)&flag != 0
}

func testSetStructAttributeDelta(k keeperlib.Keeper, ctx context.Context, structAttributeId string, oldAmount uint64, newAmount uint64) uint64 {
	currentAmount := k.GetStructAttribute(ctx, structAttributeId)
	var resetAmount uint64
	if oldAmount < currentAmount {
		resetAmount = currentAmount - oldAmount
	}
	amount := resetAmount + newAmount
	k.SetStructAttribute(ctx, structAttributeId, amount)
	return amount
}

// testFindProof brute-forces a nonce whose SHA-256 hash has the required leading zeros.
// hashTemplate should contain "%s" where the nonce is inserted.
func testFindProof(hashTemplate string, difficulty int) (nonce string, proof string) {
	for i := 0; ; i++ {
		nonce = strconv.Itoa(i)
		input := fmt.Sprintf(hashTemplate, nonce)
		hash := types.HashBuild(input)
		valid := true
		for j := 0; j < difficulty; j++ {
			if hash[j] != '0' {
				valid = false
				break
			}
		}
		if valid {
			return nonce, hash
		}
	}
}

// testAddValidator registers a bonded validator in the mock staking keeper.
func testAddValidator(k keeperlib.Keeper, valAddr sdk.ValAddress, tokens math.Int) {
	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	mock.AddValidator(valAddr, tokens)
}

type testGuildSetup struct {
	GuildOwner types.Player
	Reactor    types.Reactor
	Guild      types.Guild
	Substation types.Substation
}

func testCreateGuild(k keeperlib.Keeper, ctx context.Context) testGuildSetup {
	ownerAcc := sdk.AccAddress(fmt.Sprintf("guildowner%d_padding_addr", k.GetPlayerCount(ctx)))
	owner := types.Player{
		Creator:        ownerAcc.String(),
		PrimaryAddress: ownerAcc.String(),
	}
	owner = testAppendPlayer(k, ctx, owner)

	validatorAddress := sdk.ValAddress(ownerAcc.Bytes())
	reactor := testAppendReactor(k, ctx, types.Reactor{
		RawAddress: validatorAddress.Bytes(),
		Validator:  validatorAddress.String(),
	})

	reactorPermId := keeperlib.GetObjectPermissionIDBytes(reactor.Id, owner.Id)
	testPermissionAdd(k, ctx, reactorPermId, types.PermAll)

	alloc := types.Allocation{
		SourceObjectId: owner.Id,
		Controller:     owner.Id,
		Type:           types.AllocationType_static,
	}
	alloc, _ = testAppendAllocation(k, ctx, alloc, 100)

	substation, _, _ := testAppendSubstation(k, ctx, alloc, owner)
	substationPermId := keeperlib.GetObjectPermissionIDBytes(substation.Id, owner.Id)
	testPermissionAdd(k, ctx, substationPermId, types.PermAll)

	guild := k.AppendGuild(ctx, "test-endpoint", substation.Id, reactor, owner)

	owner.GuildId = guild.Id
	owner.GuildRank = 1
	k.SetPlayer(ctx, owner)

	guildObj, _ := k.GetGuild(ctx, guild.Id)
	guildObj.JoinInfusionMinimumBypassByInvite = types.GuildJoinBypassLevel_member
	guildObj.JoinInfusionMinimumBypassByRequest = types.GuildJoinBypassLevel_member
	k.SetGuild(ctx, guildObj)
	guild = guildObj

	guildPermId := keeperlib.GetObjectPermissionIDBytes(guild.Id, owner.Id)
	testPermissionAdd(k, ctx, guildPermId, types.PermAll)

	return testGuildSetup{
		GuildOwner: owner,
		Reactor:    reactor,
		Guild:      guild,
		Substation: substation,
	}
}

func testSetStructAttributeDecrement(k keeperlib.Keeper, ctx context.Context, structAttributeId string, decrementAmount uint64) uint64 {
	current := k.GetStructAttribute(ctx, structAttributeId)
	var newValue uint64
	if decrementAmount < current {
		newValue = current - decrementAmount
	}
	k.SetStructAttribute(ctx, structAttributeId, newValue)
	return newValue
}

func testSetStructAttributeIncrement(k keeperlib.Keeper, ctx context.Context, structAttributeId string, incrementAmount uint64) uint64 {
	current := k.GetStructAttribute(ctx, structAttributeId)
	newValue := current + incrementAmount
	k.SetStructAttribute(ctx, structAttributeId, newValue)
	return newValue
}
