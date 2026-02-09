package keeper

import (
	"structs/x/structs/types"
)

// GetPlayer returns a PlayerCache by ID, loading from store if not already cached.
func (cc *CurrentContext) GetPlayer(playerId string) (*PlayerCache, error) {
	if cache, exists := cc.players[playerId]; exists {
		return cache, nil
	}

   cc.players[playerId] = &PlayerCache{
               PlayerId: playerId,

               CC: cc,
               Loaded: true,
               Changed: false,
               Deleted: false,

               AnyChange: false,

               NonceAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_nonce, playerId),

               LastActionAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId),

               LoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, playerId),
               CapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, playerId),

               StructsLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, playerId),

               StoredOreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, playerId),

           }

	return cc.players[playerId], nil
}

// GetPlayerByAddress returns a PlayerCache by address, loading from store if not already cached.
func (cc *CurrentContext) GetPlayerByAddress(address string) (*PlayerCache, error) {
	playerIndex := cc.GetPlayerIndexFromAddress(address)
	if playerIndex == 0 {
		return nil, types.NewAddressValidationError(address, "not_registered")
	}

	player, err := cc.GetPlayer(GetObjectID(types.ObjectType_player, playerIndex))
	if err != nil {
		return nil, err
	}

	player.SetActiveAddress(address)
	return player, nil
}

// GetPlayerByAddress returns a PlayerCache by address, loading from store if not already cached.
func (cc *CurrentContext) GetPlayerByIndex(playerIndex uint64) (*PlayerCache, error) {
	player, err := cc.GetPlayer(GetObjectID(types.ObjectType_player, playerIndex))
	if err != nil {
		return nil, err
	}

	return player, nil
}

// RegisterPlayer registers an externally created PlayerCache with the context.
func (cc *CurrentContext) RegisterPlayer(cache *PlayerCache) {
	if cache == nil {
		return
	}
	cache.CC = cc
	cc.players[cache.PlayerId] = cache
	cc.pendingCommits = append(cc.pendingCommits, cache)
}
