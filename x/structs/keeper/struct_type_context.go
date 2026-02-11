package keeper

import (
	"structs/x/structs/types"
)

// GetStructType returns a StructType by ID, caching the result
func (cc *CurrentContext) GetStructType(structTypeId uint64) (*StructTypeCache, bool) {
	if cache, exists := cc.structTypes[structTypeId]; exists {
		return cache, true
	}

    structType, structTypeFound := cc.k.GetStructType(cc.ctx, structTypeId)
    if !structTypeFound {
        return &StructTypeCache{}, structTypeFound
    }

	cc.structTypes[structTypeId] = &StructTypeCache{
	    StructTypeId: structTypeId,
	    CC: cc,
		StructType:  structType,
	}
	return cc.structTypes[structTypeId], structTypeFound
}
