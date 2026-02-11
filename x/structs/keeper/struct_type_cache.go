package keeper

import (
	"structs/x/structs/types"
)

// structTypeCache is read-only (struct types don't change during operations).
type StructTypeCache struct {
    StructTypeId        uint64
    CC                  *CurrentContext
	StructType          types.StructType
}

func (cache *StructTypeCache) Commit() {}

func (cache *StructTypeCache) IsChanged() bool {
	return false
}

func (cache *StructTypeCache) ID() uint64 {
	return cache.StructTypeId
}


func (cache *StructTypeCache) GetStructType() types.StructType {
	return cache.StructType
}

