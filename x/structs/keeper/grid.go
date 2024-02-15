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


// GetGridQueueIDBytes returns the byte representation of the ID
func GetGridQueueIDBytes(objectType types.ObjectType, objectId uint64) []byte {
    id := fmt.Sprintf("%d-%d", objectType, objectId)
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


func (k Keeper) GetGridCascadeQueue(ctx sdk.Context) ([]results) {
}

func (k Keeper) ClearGridCascadeQueue(ctx sdk.Context) (err error) {
}

func (k Keeper) AppendGridCascadeQueue(ctx sdk.Context, id uint64) (err error) {
}

func (k Keeper) GridCascade(ctx sdk.Context) {
    // Get Queue
    // Clear Queue
    // For each Queue Item
        // GetGridAttribute(ctx, )
}