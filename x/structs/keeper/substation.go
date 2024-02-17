package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

)


// GetNextPlayerId allocate a new substation ID
func (k Keeper) GetNextSubstationId(ctx sdk.Context) uint64 {

    nextId := k.GetSubstationCount(ctx)

    k.SetSubstationCount(ctx, nextId + 1)

	return nextId
}

// GetSubstationCount get the total number of substation
func (k Keeper) GetSubstationCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.SubstationCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetSubstationCount set the total number of substation
func (k Keeper) SetSubstationCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.SubstationCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendSubstation appends a substation in the store with a new id and update the count
func (k Keeper) AppendSubstation(
	ctx sdk.Context,
	//substation types.Substation,
	playerConnectionAllocation uint64,
    player types.Player,
) (types.Substation) {
	substation := types.CreateEmptySubstation()

	// Set the ID of the appended value
    substation.SetId(k.GetNextSubstationId(ctx))

    // Setup some Substation details
    substation.SetOwner(player.Id)
    substation.SetCreator(player.Creator)
    substation.SetPlayerConnectionAllocation(playerConnectionAllocation)
    k.SubstationPermissionAdd(ctx, substation.Id, player.Id, types.SubstationPermissionAll)


    // actually commit to the store
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	appendedValue := k.cdc.MustMarshal(&substation)
	store.Set(GetObjectIDBytes(types.ObjectType_substation, substation.Id), appendedValue)


    // Cache invalidation event
    _ = ctx.EventManager().EmitTypedEvent(&types.EventSubstation{Substation: &substation})

	return substation
}

// SetSubstation set a specific substation in the store
func (k Keeper) SetSubstation(ctx sdk.Context, substation types.Substation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	b := k.cdc.MustMarshal(&substation)
	store.Set(GetObjectIDBytes(types.ObjectType_substation, substation.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventSubstation{Substation: &substation})

}

// RemoveSubstation removes a substation from the store
func (k Keeper) RemoveSubstation(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	store.Delete(GetObjectIDBytes(types.ObjectType_substation, id))

	_ = ctx.EventManager().EmitTypedEvent(&types.EventSubstationDelete{SubstationId: id})
}

// GetSubstation returns a substation from its id
func (k Keeper) GetSubstation(ctx sdk.Context, id uint64, full bool) (val types.Substation, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	b := store.Get(GetObjectIDBytes(types.ObjectType_substation, id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	if full {
		val.Load = k.SubstationGetLoad(ctx, val.Id)
		val.Energy = k.SubstationGetEnergy(ctx, val.Id)
		val.ConnectedPlayerCount = k.SubstationGetConnectedPlayerCount(ctx, val.Id)
	}

	return val, true
}

// GetAllSubstation returns all substation
func (k Keeper) GetAllSubstation(ctx sdk.Context, full bool) (list []types.Substation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Substation
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		if full {
			val.Load = k.SubstationGetLoad(ctx, val.Id)
			val.Energy = k.SubstationGetEnergy(ctx, val.Id)
			val.ConnectedPlayerCount = k.SubstationGetConnectedPlayerCount(ctx, val.Id)
		}

		list = append(list, val)
	}

	return
}



func (k Keeper) SubstationConnectPlayer(ctx sdk.Context, substation types.Substation, player types.Player) (error) {

    // If the player is already on a substation then disconnect them from it first
    if (player.SubstationId > 0) {
        k.SubstationDecrementConnectedPlayerLoad(ctx, player.SubstationId, 1)
    }

    // Connect Player to Substation
    k.SubstationIncrementConnectedPlayerLoad(ctx, substation.Id, 1)

    player.SetSubstation(substation.Id)
    k.SetPlayer(ctx, player)

    return nil

}

func (k Keeper) SubstationConnectAllocation(ctx sdk.Context, substation types.Substation, allocation types.Allocation)  (error) {

    // Check to see if already connected
    if (allocation.DestinationId == substation.Id) {
        // TODO add real error
        return nil
    }

	// Check to see if there is already a destination Substation using this.
	// Disconnect it if so
	if (allocation.DestinationId > 0) {
		_ = k.SubstationDecrementEnergy(ctx, allocation.DestinationId, allocation.Power)
		k.CascadeSubstationAllocationFailure(ctx, allocation.DestinationId)
	}

	allocation.SetDestinationId(substation.Id)
	k.SetAllocation(ctx, allocation)

    return nil
}

func (k Keeper) SubstationIsOnline(ctx sdk.Context, substationId uint64) (bool) {
    return (k.SubstationGetEnergy(ctx, substationId) >= k.SubstationGetLoad(ctx, substationId))
}