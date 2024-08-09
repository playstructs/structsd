package keeper

import (
	//"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
    //sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

)


func DefenderKeyPrefix(objectId string) []byte {
	return []byte(types.StructDefenderKey + objectId + "/")
}

// AppendStruct appends a struct in the store with a new id and update the count
func (k Keeper) SetStructDefender(
	ctx context.Context,
	//struct types.Struct,
	protectedStruct types.Struct,
	defendingStruct types.Struct,
) (structDefender types.StructDefender) {
 	ctxSDK := sdk.UnwrapSDKContext(ctx)

    currentProtectedStructIndex := k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, defendingStruct.Id))
    if (currentProtectedStructIndex > 0 && currentProtectedStructIndex != protectedStruct.Index) {
        // Call Remove instead of Clear since there is no reason remove this Struct Attribute, we'll update it later instead.
        k.RemoveStructDefender(ctx, GetObjectID(types.ObjectType_struct, currentProtectedStructIndex), defendingStruct.Id)
    }

 	// Get the Counter Attributes from the Type
 	defenderStructType, _ := k.GetStructType(ctx, defendingStruct.Type)

    structDefender = types.StructDefender{
          ProtectedStructId: protectedStruct.Id,
          DefendingStructId: defendingStruct.Id,

          LocationId: defendingStruct.LocationId,
          OperatingAmbit: defendingStruct.OperatingAmbit,

          CounterAttack: defenderStructType.CounterAttack,
          CounterAttackSameAmbit: defenderStructType.CounterAttackSameAmbit,
    }

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), DefenderKeyPrefix(protectedStruct.Id))
	appendedValue := k.cdc.MustMarshal(&structDefender)
	store.Set([]byte(defendingStruct.Id), appendedValue)

	// Set the Defending Structs' local attribute too
	k.SetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, defendingStruct.Id), protectedStruct.Index)

    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStructDefender{StructDefender: &structDefender})

	return structDefender
}


// GetStructDefender returns a struct defensive posture from its combo of IDs
// This function shouldn't really get used often but keeping it here for debugging
func (k Keeper) GetStructDefender(ctx context.Context, protectedStructId string, structDefenderId string) (val types.StructDefender, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), DefenderKeyPrefix(protectedStructId))
	b := store.Get([]byte(structDefenderId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// ClearStructDefender clears a structs defensive posture
func (k Keeper) ClearStructDefender(ctx context.Context, protectedStructId string, structDefenderId string) {
	k.RemoveStructDefender(ctx, protectedStructId, structDefenderId)
	k.ClearStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structDefenderId))
}


// RemoveStructDefender removes a struct defender from the store
func (k Keeper) RemoveStructDefender(ctx context.Context, protectedStructId string, structDefenderId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), DefenderKeyPrefix(protectedStructId))
	store.Delete([]byte(structDefenderId))
}

// GetAllStructDefender returns all struct defenders for a specific struct
func (k Keeper) GetAllStructDefender(ctx context.Context, protectedStructId string) (list []types.Struct) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), DefenderKeyPrefix(protectedStructId))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Struct
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

