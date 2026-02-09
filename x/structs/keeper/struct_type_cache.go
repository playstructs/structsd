package keeper

import (
	"structs/x/structs/types"
)

// structTypeCache is read-only (struct types don't change during operations).
type StructTypeCache struct {
	value  types.StructType
	loaded bool
}



