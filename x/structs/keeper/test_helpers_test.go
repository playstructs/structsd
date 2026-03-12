package keeper_test

import (
	"context"
	"fmt"

	keeperlib "structs/x/structs/keeper"
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
	k.SetPlanetCount(ctx, count+1)
	k.SetPlanet(ctx, planet)
	return planet
}

func testAppendFleet(k keeperlib.Keeper, ctx context.Context, fleet types.Fleet) types.Fleet {
	fleet.Id = fmt.Sprintf("%d-fleet-%s", types.ObjectType_fleet, fleet.Owner)
	k.SetFleet(ctx, fleet)
	return fleet
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
