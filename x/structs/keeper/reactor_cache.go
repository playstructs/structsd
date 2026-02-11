package keeper

import (
	"structs/x/structs/types"
)

type ReactorCache struct {
	ReactorId string
	CC  *CurrentContext

	ReactorLoaded bool
	Reactor         types.Reactor

	Changed bool

}

func (cache *ReactorCache) Commit() {
    if cache.Changed {

    	cache.CC.k.logger.Info("Updating Reactor From Cache", "reactorId", cache.ID())

    	if cache.Deleted {
    	    cache.CC.k.RemoveReactor(cache.CC.ctx, cache.ID())
    	} else {
    		cache.CC.k.SetReactor(cache.CC.ctx, cache.Reactor)
    	}
        cache.Changed = false
    }
}

func (cache *ReactorCache) IsChanged() bool {
	return cache.Changed
}

func (cache *ReactorCache) ID() string {
	return cache.ReactorId
}

func (cache *ReactorCache) LoadReactor() bool {
	cache.Reactor, cache.ReactorLoaded := cache.CC.k.GetReactor(cache.CC.ctx, cache.ReactorId)

	return cache.ReactorLoaded
}

func (cache *ReactorCache) CheckReactor() error {
	if !cache.Loaded {
		if !cache.LoadReactor() {
            return types.NewObjectNotFoundError("reactor", cache.ReactorId)
        }
 	}
    return nil
}

func (cache *ReactorCache) GetReactor() types.Reactor {
	if !cache.Loaded {
		cache.LoadReactor()
	}
	return cache.Reactor
}