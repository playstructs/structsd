package keeper

import (
    "structs/x/structs/types"
    "cosmossdk.io/math"
)


/* GetInfusionById resolves a stored infusion key, and reports whether it could.
 *
 * A key that does not split into three parts has no destination and no address
 * to build a cache around, so what came back was a zero InfusionCache with a nil
 * CurrentContext - and the first method called on it dereferenced that nil. The
 * caller could not tell the difference, because the old signature had nothing to
 * tell them with.
 *
 * That mattered because the only caller that reads keys off disk is the
 * destruction queue, which runs in the EndBlocker: a single malformed row there
 * panics the block, the panic discards the delete that would have removed it,
 * and the next block does it again.
 */
func (cc *CurrentContext) GetInfusionById(infusionKey string) (*InfusionCache, bool) {

    if cache, exists := cc.infusions[infusionKey]; exists {
        return cache, true
    }

    destinationId, address, err := types.ParseInfusionKey(infusionKey)
    if err != nil {
        return &InfusionCache{}, false
    }

    cc.infusions[infusionKey] = &InfusionCache{
        InfusionId:                     infusionKey,
        DestinationId:                  destinationId,
        Address:                        address,
        CC:                             cc,
        DestinationFuelAttributeId:     GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, destinationId),
        DestinationCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
    }

    return cc.infusions[infusionKey], true
}


func (cc *CurrentContext) GetInfusion(destinationId string, address string) *InfusionCache {
    infusionKey := destinationId + "-" + address

    if cache, exists := cc.infusions[infusionKey]; exists {
        return cache
    }

    cc.infusions[infusionKey] = &InfusionCache{
        InfusionId:                     infusionKey,
        DestinationId:                  destinationId,
        Address:                        address,
        CC:                             cc,
        DestinationFuelAttributeId:     GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, destinationId),
        DestinationCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
    }

    return cc.infusions[infusionKey]
}

func (cc *CurrentContext) GenesisImportInfusion(infusion types.Infusion) {
	if infusion.DestinationType == types.ObjectType_reactor {
		return
	}
	if infusion.Power == 0 {
		return
	}

	cache := cc.GetInfusion(infusion.DestinationId, infusion.Address)
	cache.Infusion = infusion
	cache.InfusionLoaded = true
	cache.Changed = true

	cache.Infusion.Recalculate()
	cache.applyGridDeltas(0, 0, 0)
}

func (cc *CurrentContext) GetAllInfusionByDestination(destinationId string) (infusions []*InfusionCache) {
    infusionIds := cc.k.GetAllInfusionIdsByDestination(cc.ctx, destinationId)
    for _, infusionId := range infusionIds {
        infusion, found := cc.GetInfusionById(infusionId)
        if !found {
            // The index is written beside the record, so a key here that cannot
            // be parsed is corrupt rather than merely absent. Skip it; the
            // destruction queue below is what clears such rows.
            cc.k.logger.Error("Unparseable infusion key in destination index", "destinationId", destinationId, "infusionKey", infusionId)
            continue
        }
        infusions = append(infusions, infusion)
    }
    return
}

// UpsertInfusion returns the infusion for a (destination, address) pair,
// creating it if it does not exist yet.
//
// Every caller resolves playerId from the address itself, so it is always that
// address's current owner. An existing record is therefore re-homed rather than
// left alone: the address may have been revoked and re-registered to somebody
// else since the record was written, and only PlayerId can drift that way.
// DestinationId and Address are the cache key, and DestinationType is implied by
// the destinationId prefix, so none of those three can disagree without corrupt
// state.
func (cc *CurrentContext) UpsertInfusion(destinationType types.ObjectType, destinationId string, address string, playerId string) (*InfusionCache){
    infusion := cc.GetInfusion(destinationId, address)

    if infusion.CheckInfusion() != nil {
        infusion.Infusion = types.Infusion{
             DestinationId:      destinationId,
             DestinationType:    destinationType,
             Address:            address,
             PlayerId:           playerId,
             Commission:         math.LegacyZeroDec(),
         }

         infusion.InfusionLoaded = true
         infusion.Changed = true
    } else if infusion.GetInfusion().PlayerId != playerId {
        infusion.SetPlayerId(playerId)
    }
    return infusion
}


func (cc *CurrentContext) ProcessInfusionDestructionQueue() {
    for {
        queue := cc.k.GetInfusionDestructionQueue(cc.ctx, true)
        if len(queue) == 0 {
            break
        }

        for _, infusionId := range queue {
            infusion, found := cc.GetInfusionById(infusionId)
            if !found {
                /* A key that cannot be parsed has no infusion behind it and
                 * never will. Dropping it is the whole repair: this runs in the
                 * EndBlocker, and calling through to the cache would dereference
                 * a nil CurrentContext and panic the block - taking the delete
                 * above down with it, so the same row would be read again next
                 * block, and every block after that.
                 *
                 * The read already cleared the row, so returning without
                 * re-queueing is what discards it.
                 */
                cc.k.logger.Error("Discarding unparseable infusion destruction queue entry", "infusionKey", infusionId)
                continue
            }

            if (infusion.CheckInfusion() == nil && infusion.IsEmpty()) {
                infusion.Destroy()
            }
        }
    }
}

func (cc *CurrentContext) DestroyAllInfusions(infusionIds []string) {
	for _, infusionId := range infusionIds {
		infusion, found := cc.GetInfusionById(infusionId)
		if !found {
			cc.k.logger.Error("Skipping unparseable infusion key", "infusionKey", infusionId)
			continue
		}
		if infusion.CheckInfusion() == nil {
		    infusion.Destroy()
		}
	}
}