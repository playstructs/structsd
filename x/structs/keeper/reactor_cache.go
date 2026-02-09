package keeper

import (
	"structs/x/structs/types"
)

type ReactorCache struct {
	value   types.Reactor
	loaded  bool
	changed bool
}

