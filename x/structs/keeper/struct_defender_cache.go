package keeper

import (
	"structs/x/structs/types"
)

// structDefenderCache holds a struct defender relationship with change tracking.
type StructDefenderCache struct {
    id      string
	value   types.StructDefender
	loaded  bool
	changed bool
}