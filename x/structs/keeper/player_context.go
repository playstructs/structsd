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


func (cc *CurrentContext) GetAllPlayerBySubstation(substationId string) (players []*PlayerCache) {
    playerList := cc.k.GetAllPlayerIdBySubstationIndex(cc.ctx, substationId)

    for _, playerId := range playerList {
        player := cc.GetPlayer(playerId)
        players = append(players, player)
    }
    return
}

func (cc *CurrentContext) NewPlayer(address string) *PlayerCache {

	// Create the player
    var player types.Player

    player.Index = k.GetPlayerCount(ctx)
	cc.k.SetPlayerCount(cc.ctx, player.Index + 1)

	playerId := GetObjectID(types.ObjectType_player, player.Index)
	player.Id = playerId

	player.Creator = address
	player.PrimaryAddress = address

    cc.players[playerId] = &PlayerCache{
               PlayerId: playerId,

               CC: cc,
               Player: player,
               PlayerLoaded: true,
               Changed: true,
               Deleted: false,

               NonceAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_nonce, playerId),

               LastActionAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId),

               LoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, playerId),
               CapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, playerId),

               StructsLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, playerId),

               StoredOreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, playerId),
           }

	//Add Address records
	cc.k.SetPlayerIndexForAddress(cc.ctx, player.Creator, player.Index)

    //Add permissions
	addressPermissionId := GetAddressPermissionIDBytes(player.Creator)
	cc.PermissionAdd(addressPermissionId, types.PermissionAll)

    // Add the initial Player Load
    cc.SetGridAttributeIncrement(cc.players[playerId].StructsLoadAttributeId, types.PlayerPassiveDraw)

	return cc.players[playerId]
}

// Technically more of an InGet than an UpSert
func (cc *CurrentContext) UpsertPlayer(address string) (player *PlayerCache) {
    playerIndex := cc.k.GetPlayerIndexFromAddress(cc.ctx, playerAddress)

    if (playerIndex == 0) {
        player = cc.NewPlayer(address)
    } else {
        player, _ = cc.GetPlayerByIndex(playerIndex)
    }

    return
}

