package keeper

import (
	"structs/x/structs/types"
)

// GetReactor returns a Reactor by ID, caching the result.
func (cc *CurrentContext) GetReactor(reactorId string) (*ReactorCache) {
	if cache, exists := cc.reactors[reactorId]; exists {
		return cache
	}

	cc.reactors[reactorId] = &ReactorCache{
	    ReactorId: reactorId,
	    CC: cc,
	    ReactorLoaded: false,
	    Changed: false,
	}

	return cc.reactors[reactorId]
}

