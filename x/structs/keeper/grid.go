package keeper

import (
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"fmt"
)



// GetGridAttributeIDBytes returns the byte representation of the ID
func GetGridAttributeIDBytes(gridAttributeType types.GridAttributeType, objectType types.ObjectType, objectId uint64) []byte {
    id := fmt.Sprintf("%d-%d-%d", gridAttributeType, objectType, objectId)
	return []byte(id)
}

// GetGridAttributeIDBytesByGridQueueId returns the byte representation of the ID
func GetGridAttributeIDBytesByGridQueueId(gridAttributeType types.GridAttributeType, gridQueueId []byte) []byte {
    id := fmt.Sprintf("%d-%s", gridAttributeType, string(gridQueueId))
	return []byte(id)
}



func (k Keeper) GetGridAttribute(ctx sdk.Context, gridAttributeId []byte) (amount uint64, err error) {

}


func (k Keeper) SetGridAttribute(ctx sdk.Context, gridAttributeId []byte, amount uint64) (err error) {

}

func (k Keeper) SetGridAttributeDelta(ctx sdk.Context, gridAttributeId []byte, oldAmount uint64, newAmount uint64) (amount uint64, err error) {

}

func (k Keeper) SetGridAttributeDecrement(ctx sdk.Context, gridAttributeId []byte, decrementAmount uint64) (amount uint64, err error) {

}

func (k Keeper) SetGridAttributeIncrement(ctx sdk.Context, gridAttributeId []byte, incrementAmount uint64) (amount uint64, err error) {

}

// GetGridQueueIDBytes returns the byte representation of the ID
func GetGridQueueIDBytes(objectType types.ObjectType, objectId uint64) []byte {
    id := fmt.Sprintf("%d-%d", objectType, objectId)
	return []byte(id)
}


func (k Keeper) GetGridCascadeQueue(ctx sdk.Context, clear bool) (queue [][]byte) {
	gridCascadeQueueStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GridCascadeQueue))
	iterator := sdk.KVStorePrefixIterator(gridCascadeQueueStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		queue = append(queue, iterator.Key())
		if clear {
		    gridCascadeQueueStore.Delete(iterator.Key())
		}
	}

    return
}


func (k Keeper) AppendGridCascadeQueue(ctx sdk.Context, queueId []byte) (err error) {
    gridCascadeQueueStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.GridCascadeQueue))

	bz := make([]byte, 1)
	binary.BigEndian.PutBool(bz, bool(true))

	gridCascadeQueueStore.Set(queueId, bz)

	return
}

func (k Keeper) GridCascade(ctx sdk.Context) {

    // Get Queue (and clear it in the process)
    gridQueue := GetGridCascadeQueue(ctx, true)

    // For each Queue Item
    for _, queueId := range gridQueue {
        // GetGridAttribute(ctx, )

    }

}