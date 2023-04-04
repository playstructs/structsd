package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

// GetSubstationCount get the total number of substation
func (k Keeper) GetSubstationCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.SubstationCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
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
	substation types.Substation,
) uint64 {
	// Create the substation
	count := k.GetSubstationCount(ctx)

	// Set the ID of the appended value
	substation.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	appendedValue := k.cdc.MustMarshal(&substation)
	store.Set(GetSubstationIDBytes(substation.Id), appendedValue)

	// Update substation count
	k.SetSubstationCount(ctx, count+1)

	return count
}

// SetSubstation set a specific substation in the store
func (k Keeper) SetSubstation(ctx sdk.Context, substation types.Substation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	b := k.cdc.MustMarshal(&substation)
	store.Set(GetSubstationIDBytes(substation.Id), b)
}

// GetSubstation returns a substation from its id
func (k Keeper) GetSubstation(ctx sdk.Context, id uint64) (val types.Substation, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	b := store.Get(GetSubstationIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSubstation removes a substation from the store
func (k Keeper) RemoveSubstation(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	store.Delete(GetSubstationIDBytes(id))
}

// GetAllSubstation returns all substation
func (k Keeper) GetAllSubstation(ctx sdk.Context) (list []types.Substation) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubstationKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Substation
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetSubstationIDBytes returns the byte representation of the ID
func GetSubstationIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetSubstationIDFromBytes returns ID in uint64 format from a byte array
func GetSubstationIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
