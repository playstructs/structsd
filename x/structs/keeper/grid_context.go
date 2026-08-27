package keeper

import (
    "math"
    "strings"

    "structs/x/structs/types"
)


// GetGridAttribute returns a grid attribute value, caching the result.
func (cc *CurrentContext) GetGridAttribute(gridAttributeId string) uint64 {
	if cache, exists := cc.gridAttributes[gridAttributeId]; exists {
		return cache.Value
	}

	value := cc.k.GetGridAttribute(cc.ctx, gridAttributeId)
	cc.gridAttributes[gridAttributeId] = &GridAttributeCache{
	    CC:     cc,
	    GridAttributeId: gridAttributeId,
	    Value:  value,
	    Loaded: true,
    }
	return value
}

func (cc *CurrentContext) SetGridAttribute(gridAttributeId string, value uint64) {
	cc.gridAttributes[gridAttributeId] = &GridAttributeCache{
 	    CC:     cc,
 	    GridAttributeId: gridAttributeId,
	    Value: value,
	    Loaded: true,
	    Deleted: false,
	    Changed: true,
	}
}

func (cc *CurrentContext) ClearGridAttribute(gridAttributeId string) {
	cc.gridAttributes[gridAttributeId] = &GridAttributeCache{
 	    CC:                 cc,
 	    GridAttributeId:    gridAttributeId,
	    Value: 0,
	    Loaded: true,
	    Changed: true,
	    Deleted: true,
	}
}

/* SetGridAttributeIncrement adds to a grid attribute, saturating at MaxUint64.
 *
 * The addition was unguarded while both of its siblings guarded their
 * subtraction, which made it the odd one out rather than a considered choice.
 * Wrapping is the worst of the three outcomes available here: a capacity or a
 * stored-ore balance that rolls over to a small number destroys accounted value
 * and understates committed load, and every caller treats these attributes as
 * monotonic.
 *
 * Saturating rather than returning an error is deliberate. Most callers cannot
 * refuse - genesis import, staking hooks and the block hooks have nowhere to
 * put a rejection - and an error they would have to drop is a worse contract
 * than a bound they cannot exceed. The log is what makes it diagnosable; a
 * saturated attribute is corrupt state either way, and the point of the clamp is
 * that it is corrupt in a bounded, deterministic direction instead of a
 * wrapped one.
 *
 * Reaching the bound is not currently possible: the only compounding path is a
 * cycle of allocations feeding each other, which grows linearly rather than
 * exponentially. That cycle is its own bug and is tracked separately - this is
 * the backstop, not the fix for it.
 */
func (cc *CurrentContext) SetGridAttributeIncrement(gridAttributeId string, delta uint64) uint64 {
	current := cc.GetGridAttribute(gridAttributeId)

	newValue := current + delta
	if newValue < current {
		cc.k.logger.Error("Grid attribute increment saturated",
			"gridAttributeId", gridAttributeId,
			"current", current,
			"delta", delta,
		)
		newValue = math.MaxUint64
	}

	cc.SetGridAttribute(gridAttributeId, newValue)
	return newValue
}

// SetGridAttributeDecrement decrements a grid attribute
// Will not go below zero.
func (cc *CurrentContext) SetGridAttributeDecrement(gridAttributeId string, delta uint64) uint64 {
	current := cc.GetGridAttribute(gridAttributeId)
	var newValue uint64
	if delta < current {
		newValue = current - delta
	}
	cc.SetGridAttribute(gridAttributeId, newValue)
	return newValue
}

// Updates a Grid Attribute by first removing the old amount and then adding the new amount
func (cc *CurrentContext) SetGridAttributeDelta(gridAttributeId string, oldAmount uint64, newAmount uint64) (uint64) {
	currentAmount := cc.GetGridAttribute(gridAttributeId)

	var resetAmount uint64
	if oldAmount < currentAmount {
		resetAmount = currentAmount - oldAmount
	}

	// The subtraction above was guarded and this addition was not, which is the
	// same gap SetGridAttributeIncrement carried. Saturate for the same reasons.
	amount := resetAmount + newAmount
	if amount < resetAmount {
		cc.k.logger.Error("Grid attribute delta saturated",
			"gridAttributeId", gridAttributeId,
			"current", currentAmount,
			"oldAmount", oldAmount,
			"newAmount", newAmount,
		)
		amount = math.MaxUint64
	}

    cc.SetGridAttribute(gridAttributeId, amount)

	return amount
}

func (cc *CurrentContext) UpdateSubstationConnectionCapacity(objectId string) {
    if strings.HasPrefix(objectId, "4-") {

        capacityAttributeId             := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)
        loadAttributeId                 := GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)
        connectionCapacityAttributeId   := GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId)
        connectionCountAttributeId      := GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, objectId)

        capacity    := cc.GetGridAttribute(capacityAttributeId)
        load        := cc.GetGridAttribute(loadAttributeId)

        if capacity > load {
            availableCapacity := capacity - load

            connectionCount := cc.GetGridAttribute(connectionCountAttributeId)
            if connectionCount == 0 {
                connectionCount = 1
            }

            cc.SetGridAttribute(connectionCapacityAttributeId, availableCapacity/connectionCount)
        } else {
            cc.SetGridAttribute(connectionCapacityAttributeId, 0)
        }
    }
}




/* GridCascade sheds load from every object that is over-subscribed, and stops
 * when it runs out of budget rather than when it runs out of work.
 *
 * The queue is drained in sequence order up to GridCascadeBlockBudget
 * allocations destroyed. What is left stays queued and is picked up by the next
 * block, ahead of anything enqueued since - that ordering is what stops an
 * attacker from parking their own over-subscribed substation at the back of the
 * line indefinitely. See the GridCascadeQueue comment in types/keys.go.
 *
 * A deferred object is not a granted one. Every gate that sells power compares
 * before it subtracts - SubstationCache.GetAvailableCapacity,
 * PlayerCache.GetAvailableCapacity, CanSupportLoadAddition - so an
 * over-subscribed object reports zero headroom and can hand out nothing new
 * while it waits. What it keeps doing is powering what is already attached,
 * which is the same "one last block of power" this has always accepted, over
 * more blocks.
 */
func (cc *CurrentContext) GridCascade() {

	budget := types.GridCascadeBlockBudget

	for budget > 0 {
		// Read without clearing: an entry is removed when it is finished, so
		// whatever the budget does not reach is still queued next block.
		batch := cc.k.GetGridCascadeQueueBatch(cc.ctx, budget)

		if len(batch) == 0 {
			break
		}

		for _, entry := range batch {
			if budget <= 0 {
				break
			}

			objectId := entry.ObjectId

			// Visiting an entry costs budget even when it turns out to need no
			// work. Charging only for destroys would leave the walk itself
			// unbounded, and a queue of objects that are each already under
			// capacity is exactly as cheap to build and exactly as expensive to
			// walk.
			budget--

			// Clear the entry before doing the work. The cascade below can
			// re-enqueue this same object through Destroy, and that append has
			// to be able to take a fresh sequence rather than be swallowed as a
			// duplicate of the row being processed.
			cc.k.RemoveGridCascadeQueueEntry(cc.ctx, entry)

			if objectId == "" {
				continue
			}

			// Only as many allocations as this block could possibly shed. The
			// source's fan-out is attacker-sized - capacity can be split into
			// one-power allocations - and loading the whole set to destroy a
			// handful of them puts the unbounded work back in the EndBlocker
			// that the budget was supposed to take out of it.
			allocationList, moreAllocations := cc.GetAllocationsBySourceUpTo(objectId, budget)
			allocationPointer := 0

			loadAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)
			capacityAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)

			for cc.GetGridAttribute(loadAttributeId) > cc.GetGridAttribute(capacityAttributeId) {
				if allocationPointer >= len(allocationList) {
					if moreAllocations {
						// Not out of allocations, out of the batch. Requeue so
						// the rest are shed next block; this is the same
						// deferral as running out of budget below, reached by
						// the other road.
						if err := cc.k.AppendGridCascadeQueue(cc.ctx, objectId); err != nil {
							cc.k.logger.Error("Grid Queue (requeue failed)", "objectId", objectId, "error", err)
						}
						break
					}

					// Genuinely nothing left to shed and still over capacity.
					// Something is probably wrong here...
					cc.k.logger.Warn("Grid Queue problem", "objectId", objectId)
					break
				}

				if budget <= 0 {
					// Out of budget with this object still over-subscribed.
					// Re-queue it so the remaining allocations are shed next
					// block; it goes to the back, but nothing that arrives
					// later can overtake it.
					if err := cc.k.AppendGridCascadeQueue(cc.ctx, objectId); err != nil {
						cc.k.logger.Error("Grid Queue (requeue failed)", "objectId", objectId, "error", err)
					}
					break
				}

				cc.k.logger.Info("Grid Queue (Brownout)", "objectId", objectId, "load", cc.GetGridAttribute(loadAttributeId), "capacity", cc.GetGridAttribute(capacityAttributeId))

				// The brownout has to keep shedding load even if settling an
				// agreement behind one of these allocations fails, or the loop
				// spins on a source it can never bring back under capacity.
				if err := allocationList[allocationPointer].Destroy(); err != nil {
					cc.k.logger.Error("Grid Queue (Allocation Destroy failed)", "allocationId", allocationList[allocationPointer].GetAllocationId(), "error", err)
				}
				cc.k.logger.Info("Grid Queue (Allocation Destroyed)", "allocationId", allocationList[allocationPointer].GetAllocationId())

				allocationPointer++
				budget--
			}
		}
	}

	// Never leave a cap silent: a backlog here means objects are running
	// over-subscribed until a later block reaches them.
	if remaining := cc.k.GetGridCascadeQueueLength(cc.ctx); remaining > 0 {
		cc.k.logger.Warn("Grid Queue (deferred)", "remaining", remaining, "budget", types.GridCascadeBlockBudget)
	}
}