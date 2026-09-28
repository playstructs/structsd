package keeper

import "sort"

// indexCachedOwner keeps an owner-scoped overlay of loaded, live infusions.
// Probes for absent rows never enter it. It holds the canonical cache pointers;
// only cc.infusions commits them.
func (cache *InfusionCache) indexCachedOwner() {
	owner := ""
	if cache.InfusionLoaded && !cache.Deleted {
		owner = cache.Infusion.AssetOwnerID()
	}
	if cache.cachedOwnerId != "" && cache.cachedOwnerId != owner {
		delete(cache.CC.infusionsByOwner[cache.cachedOwnerId], cache.InfusionId)
	}
	if owner != "" {
		if cache.CC.infusionsByOwner[owner] == nil {
			cache.CC.infusionsByOwner[owner] = make(map[string]*InfusionCache)
		}
		cache.CC.infusionsByOwner[owner][cache.InfusionId] = cache
	}
	cache.cachedOwnerId = owner
}

// GetInfusionsByOwnerUpTo resolves an entire owner inventory or returns complete
// false and no inventory. The limit counts this owner's raw index rows plus
// uncommitted additions, once each. Other owners and duplicate cache hits never
// consume its allowance. Ownership changes and deletion are reflected immediately.
// Eligibility is deliberately re-evaluated by callers against live staking state.
func (cc *CurrentContext) GetInfusionsByOwnerUpTo(owner string, limit int) (infusions []*InfusionCache, complete bool, workUsed int) {
	if limit < 0 || len(cc.infusionsByOwner[owner]) > limit {
		return nil, false, 0
	}
	entries, more, rows := cc.k.GetInfusionOwnerEntriesUpTo(cc.ctx, owner, limit)
	workUsed = rows
	if more || len(entries) != rows {
		return nil, false, workUsed
	}
	indexed := make(map[string]bool, len(entries))
	resolved := make(map[string]*InfusionCache, len(entries))
	for _, entry := range entries {
		cache := cc.GetInfusion(entry.DestinationId, entry.Address)
		indexed[cache.InfusionId] = true
		// A row moved/deleted in this operation is explained only by its
		// original committed owner. An unrelated stale index remains invalid.
		if cache.Deleted || (cache.InfusionLoaded && cache.Infusion.AssetOwnerID() != owner) {
			if cache.originalOwnerKnown && cache.originalOwnerId == owner {
				continue
			}
			return nil, false, workUsed
		}
		if cache.CheckInfusion() != nil || cache.Infusion.AssetOwnerID() != owner {
			return nil, false, workUsed
		}
		resolved[cache.InfusionId] = cache
	}
	// This map includes only this owner's loaded live rows. Sort before work
	// that can fail so iteration order never selects a different partial result.
	keys := make([]string, 0, len(cc.infusionsByOwner[owner]))
	for key := range cc.infusionsByOwner[owner] {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !indexed[key] {
			if workUsed >= limit {
				return nil, false, workUsed
			}
			workUsed++
		}
		resolved[key] = cc.infusionsByOwner[owner][key]
	}
	keys = keys[:0]
	for key := range resolved {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		infusions = append(infusions, resolved[key])
	}
	return infusions, true, workUsed
}
