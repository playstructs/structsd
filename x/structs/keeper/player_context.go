package keeper

import (
	"structs/x/structs/types"
)

// GetPlayer returns a PlayerCache by ID. Like every other context getter it is a
// cache allocator: it does not read the store, so it cannot report whether the
// player exists and deliberately has no error return. A caller handed an id by a
// transaction wants GetExistingPlayer instead — a cache for a player that is not
// there loads as a zero value, silently drops writes, because a failed load
// leaves PlayerLoaded false and every getter reloads over them, and commits to
// the empty key, which the KV store panics on.
func (cc *CurrentContext) GetPlayer(playerId string) *PlayerCache {
	if cache, exists := cc.players[playerId]; exists {
		return cache
	}

   cc.players[playerId] = &PlayerCache{
               PlayerId: playerId,

               CC: cc,
               PlayerLoaded: false,
               Changed: false,
               Deleted: false,

               NonceAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_nonce, playerId),

               LastActionAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId),

               LoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, playerId),
               CapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, playerId),

               StructsLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, playerId),

               StoredOreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, playerId),

           }

	return cc.players[playerId]
}

// GetExistingPlayer returns a PlayerCache only if that player is in state. Use it
// for any id a message supplies: GetPlayer cannot fail, so an `if err != nil`
// guard on it is dead code that reads like an existence check.
func (cc *CurrentContext) GetExistingPlayer(playerId string) (*PlayerCache, error) {
	player := cc.GetPlayer(playerId)
	if err := player.CheckPlayer(); err != nil {
		return nil, err
	}

	return player, nil
}

func (cc *CurrentContext) GenesisImportPlayer(player types.Player) {
	cache := cc.GetPlayer(player.Id)
	cache.Player = player
	cache.PlayerLoaded = true
	cache.Changed = true

	cc.SetGridAttribute(cache.StructsLoadAttributeId, types.PlayerPassiveDraw)
	cc.SetGridAttribute(cache.LastActionAttributeId, 0)

	if player.SubstationId != "" {
		cc.k.SetSubstationPlayerIndex(cc.ctx, player.SubstationId, player.Id)
		substation := cc.GetSubstation(player.SubstationId)
		cc.SetGridAttributeIncrement(substation.ConnectionCountAttributeId, 1)
	}
}

// GetSigningPlayer returns the PlayerCache for the authenticated signer and
// records that address as the acting identity for this operation. Handlers must
// use this for msg.Creator and only for msg.Creator: every Layer 1 permission
// check resolves against the recorded address, so passing a caller-supplied
// address here would let a weak key borrow a stronger one's bits.
func (cc *CurrentContext) GetSigningPlayer(address string) (*PlayerCache, error) {
	player, err := cc.GetPlayerByAddress(address)
	if err != nil {
		return nil, err
	}

	if err := cc.setSigner(address); err != nil {
		return nil, err
	}

	return player, nil
}

// GetPlayerByAddress returns the PlayerCache owning an address, loading from
// store if not already cached. This is a pure lookup: it does not establish who
// is acting. PlayerCache instances are shared by every address of a player, so
// there is nothing per-address about the value returned.
func (cc *CurrentContext) GetPlayerByAddress(address string) (*PlayerCache, error) {
	playerIndex := cc.GetPlayerIndexFromAddress(address)
	if playerIndex == 0 {
		return nil, types.NewAddressValidationError(address, "not_registered")
	}

	return cc.GetPlayer(GetObjectID(types.ObjectType_player, playerIndex)), nil
}

// GetPlayerByIndex returns a PlayerCache by index. Allocator semantics, as GetPlayer.
func (cc *CurrentContext) GetPlayerByIndex(playerIndex uint64) *PlayerCache {
	return cc.GetPlayer(GetObjectID(types.ObjectType_player, playerIndex))
}


func (cc *CurrentContext) GetAllPlayerBySubstation(substationId string) (players []*PlayerCache) {
    playerList := cc.k.GetAllPlayerIdBySubstationIndex(cc.ctx, substationId)

    for _, playerId := range playerList {
        players = append(players, cc.GetPlayer(playerId))
    }
    return
}

func (cc *CurrentContext) NewPlayer(address string) *PlayerCache {

	// Create the player
    var player types.Player

    player.Index = cc.k.GetPlayerCount(cc.ctx)
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
	//
	// Every caller reaches here with an address that is already known-good: a
	// handler's SDK-validated msg.Creator, an address checked against
	// PubKeyToBech32 by a proof, or an AccAddress rendered back to a string by
	// the staking hooks. So this cannot fire, and it is logged rather than
	// propagated because NewPlayer has no error return and returning nil here
	// would hand every caller a nil cache to dereference — a worse failure than
	// the one being guarded.
	if err := cc.k.SetPlayerIndexForAddress(cc.ctx, player.Creator, player.Index); err != nil {
		cc.k.logger.Error("New player created with an address that could not be indexed", "playerId", playerId, "address", player.Creator, "error", err)
	}

    //Add permissions
	addressPermissionId := GetAddressPermissionIDBytes(player.Creator)
	cc.SetPermissions(addressPermissionId, types.PermAll)

    playerPermissionId := GetObjectPermissionIDBytes(playerId, playerId)
    cc.SetPermissions(playerPermissionId, types.PermAll)


    // Add the initial Player Load
    cc.SetGridAttributeIncrement(cc.players[playerId].StructsLoadAttributeId, types.PlayerPassiveDraw)

	return cc.players[playerId]
}

// Technically more of an InGet than an UpSert
func (cc *CurrentContext) UpsertPlayer(address string) (player *PlayerCache) {
    playerIndex := cc.k.GetPlayerIndexFromAddress(cc.ctx, address)

    if (playerIndex == 0) {
        player = cc.NewPlayer(address)
    } else {
        player = cc.GetPlayerByIndex(playerIndex)
    }

    return
}

