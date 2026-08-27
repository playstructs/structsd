package keeper

import (
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

// SetGridAttributeIncrement increments a grid attribute
func (cc *CurrentContext) SetGridAttributeIncrement(gridAttributeId string, delta uint64) uint64 {
	current := cc.GetGridAttribute(gridAttributeId)
	newValue := current + delta
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
	amount := resetAmount + newAmount

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

			allocationList := cc.GetAllAllocationBySource(objectId)
			allocationPointer := 0

			loadAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)
			capacityAttributeId := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)

			for cc.GetGridAttribute(loadAttributeId) > cc.GetGridAttribute(capacityAttributeId) {
				if allocationPointer >= len(allocationList) {
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