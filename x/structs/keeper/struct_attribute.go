package keeper

import (
	"encoding/binary"
	"context"
	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"

	"fmt"
)



// GetStructAttributeID returns the string representation of the ID
func GetStructAttributeID(structAttributeType types.StructAttributeType, objectType types.ObjectType, objectId uint64) string {
    id := fmt.Sprintf("%d-%d-%d", structAttributeType, objectType, objectId)
	return id
}

// GetStructAttributeIDByObjectId returns the string representation of the ID
func GetStructAttributeIDByObjectId(structAttributeType types.StructAttributeType, objectId string) string {
    id := fmt.Sprintf("%d-%s", structAttributeType, objectId)
	return id
}

// GetStructAttributeIDByObjectIdAndSubIndex returns the string representation of the ID
func GetStructAttributeIDByObjectIdAndSubIndex(structAttributeType types.StructAttributeType, objectId string, index uint64) string {
    id := fmt.Sprintf("%d-%s-%d", structAttributeType, objectId, index)
	return id
}

func (k Keeper) GetStructAttribute(ctx context.Context, structAttributeId string) (amount uint64) {
	structAttributeStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructAttributeKey))

	bz := structAttributeStore.Get([]byte(structAttributeId))

	if bz == nil {
        // return error?
        // err =
		amount = 0
	} else {
		amount = binary.BigEndian.Uint64(bz)
	}

	return
}

func (k Keeper) ClearStructAttribute(ctx context.Context, structAttributeId string) () {
	structAttributeStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructAttributeKey))
	structAttributeStore.Delete([]byte(structAttributeId))
}


func (k Keeper) SetStructAttribute(ctx context.Context, structAttributeId string, amount uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.StructAttributeKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set([]byte(structAttributeId), bz)


	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStructAttribute{&types.StructAttributeRecord{AttributeId: structAttributeId, Value: amount}})
    fmt.Printf("Struct Change (Set): (%s) %d \n", structAttributeId, amount)
}

func (k Keeper) SetStructAttributeDelta(ctx context.Context, structAttributeId string, oldAmount uint64, newAmount uint64) (amount uint64, err error) {
    currentAmount := k.GetStructAttribute(ctx, structAttributeId)

    var resetAmount uint64
    if (oldAmount < currentAmount) {
        resetAmount = currentAmount - oldAmount
    }

    amount = resetAmount + newAmount

    fmt.Printf("Struct Change (Delta): (%s) %d to %d \n", structAttributeId, oldAmount, newAmount)
    k.SetStructAttribute(ctx, structAttributeId, amount)

    return
}

func (k Keeper) SetStructAttributeDecrement(ctx context.Context, structAttributeId string, decrementAmount uint64) (amount uint64, err error) {
    currentAmount := k.GetStructAttribute(ctx, structAttributeId)

    if (decrementAmount < currentAmount) {
        amount = currentAmount - decrementAmount
    }

    fmt.Printf("Struct Change (Decrement): (%s) %d \n", structAttributeId, decrementAmount)
    k.SetStructAttribute(ctx, structAttributeId, amount)

    return
}

func (k Keeper) SetStructAttributeIncrement(ctx context.Context, structAttributeId string, incrementAmount uint64) (amount uint64) {
    currentAmount := k.GetStructAttribute(ctx, structAttributeId)

    amount = currentAmount + incrementAmount

    fmt.Printf("Struct Change (Increment): (%s) %d \n", structAttributeId, incrementAmount)
    k.SetStructAttribute(ctx, structAttributeId, amount)

    return
}

/* The Struct Attribute Store also supports bitwise flags */

func (k Keeper) SetStructAttributeFlagAdd(ctx context.Context, structAttributeId string, flag uint64) uint64 {
    currentFlags    := k.GetStructAttribute(ctx, structAttributeId)
    newFlags        := currentFlags | flag
    k.SetStructAttribute(ctx, structAttributeId, newFlags)
	return newFlags
}

func (k Keeper) SetStructAttributeFlagRemove(ctx context.Context, structAttributeId string, flag uint64) uint64 {
    currentFlags    := k.GetStructAttribute(ctx, structAttributeId)
    newFlags        := currentFlags &^ flag
    k.SetStructAttribute(ctx, structAttributeId, newFlags)
	return newFlags
}

func (k Keeper) StructAttributeFlagHasAll(ctx context.Context, structAttributeId string, flag uint64) bool {
    currentFlags := k.GetStructAttribute(ctx, structAttributeId)
	return currentFlags&flag == flag
}

func (k Keeper) StructAttributeFlagHasOneOf(ctx context.Context, structAttributeId string, flag uint64) bool {
    currentFlags := k.GetStructAttribute(ctx, structAttributeId)
	return currentFlags&flag != 0
}



func (k Keeper) GetStructAttributesByObject(ctx context.Context, objectId string) (types.StructAttributes) {
    status := k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_status, objectId))
    return types.StructAttributes{
        Health: k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_health, objectId)),
        Status: status,

        BlockStartBuild: k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, objectId)),
        BlockStartOreMine: k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, objectId)),
        BlockStartOreRefine: k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, objectId)),

        ProtectedStructIndex: k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, objectId)),

        //typeCount: k.GetStructAttribute(ctx, GetStructAttributeIDByObjectIdAndSubIndex(types.StructAttributeType_typeCount, objectId),

        IsMaterialized:    types.StructState(status)&types.StructStateMaterialized != 0,
        IsBuilt:           types.StructState(status)&types.StructStateBuilt != 0,
        IsOnline:          types.StructState(status)&types.StructStateOnline != 0,
        IsHidden:          types.StructState(status)&types.StructStateHidden != 0,
        IsDestroyed:       types.StructState(status)&types.StructStateDestroyed != 0,
        IsLocked:          types.StructState(status)&types.StructStateLocked != 0,

  }
}