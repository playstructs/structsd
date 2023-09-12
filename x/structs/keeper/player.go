package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"

)


// GetPlayerCount get the total number of player
func (k Keeper) GetPlayerCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.PlayerCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil || binary.BigEndian.Uint64(bz) == 0 {
		return types.KeeperStartValue
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetPlayerCount set the total number of player
func (k Keeper) SetPlayerCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.PlayerCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendPlayer appends a player in the store with a new id and update the count
func (k Keeper) AppendPlayer(
	ctx sdk.Context,
	player types.Player,
) uint64 {
	// Create the player
	count := k.GetPlayerCount(ctx)

	// Set the ID of the appended value
	player.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	appendedValue := k.cdc.MustMarshal(&player)
	store.Set(GetPlayerIDBytes(player.Id), appendedValue)

	// Update player count
	k.SetPlayerCount(ctx, count+1)

	//Add Address records
	k.SetPlayerIdForAddress(ctx, player.Creator, player.Id)
	k.AddressPermissionAdd(ctx, player.Creator, types.AddressPermissionAll)

	// Add the Account keeper record
	// This is needed for the proxy account creation
	playerAccAddress, _ := sdk.AccAddressFromBech32(player.Creator)
	playerAuthAccount := k.accountKeeper.GetAccount(ctx, playerAccAddress)
    if playerAuthAccount == nil {
        playerAuthAccount = k.accountKeeper.NewAccountWithAddress(ctx, playerAccAddress)
        k.accountKeeper.SetAccount(ctx, playerAuthAccount)
    }


	_ = ctx.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})

	return count
}

// SetPlayer set a specific player in the store
func (k Keeper) SetPlayer(ctx sdk.Context, player types.Player) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	b := k.cdc.MustMarshal(&player)

	store.Set(GetPlayerIDBytes(player.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})
}

// GetPlayer returns a player from its id
func (k Keeper) GetPlayer(ctx sdk.Context, id uint64) (val types.Player, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	b := store.Get(GetPlayerIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	val.Load = k.PlayerGetLoad(ctx, val.Id)
	playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
    val.Storage = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")

	return val, true
}

// RemovePlayer removes a player from the store
func (k Keeper) RemovePlayer(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	store.Delete(GetPlayerIDBytes(id))
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayer(ctx sdk.Context) (list []types.Player) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		val.Load = k.PlayerGetLoad(ctx, val.Id)
		playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
		val.Storage = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")

		list = append(list, val)
	}

	return
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayerBySubstation(ctx sdk.Context, substationId uint64) (list []types.Player) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if (val.SubstationId == substationId) {
            val.Load = k.PlayerGetLoad(ctx, val.Id)
            playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
            val.Storage = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")

            list = append(list, val)
		}
	}

	return
}

// GetPlayerIDBytes returns the byte representation of the ID
func GetPlayerIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetPlayerIDFromBytes returns ID in uint64 format from a byte array
func GetPlayerIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}


// Technically more of an InGet than an UpSert
func (k Keeper) UpsertPlayer(ctx sdk.Context, playerAddress string ) (player types.Player) {
    playerId := k.GetPlayerIdFromAddress(ctx, playerAddress)

    if (playerId == 0) {
        // No Player Found, Creating..
        player = types.CreateEmptyPlayer()
        player.SetCreator(playerAddress)
        playerId = k.AppendPlayer(ctx, player)
        player.SetId(playerId)

    } else {
        player, _ = k.GetPlayer(ctx, playerId)
    }

    return player
}








// the current total load of the player structs
// Go to memory first, but then fall back to rebuilding from storage
func (k Keeper) PlayerGetLoad(ctx sdk.Context, id uint64) (load uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerLoadKey))

	bz := store.Get(GetPlayerIDBytes(id))

	// Substation Capacity Not in Memory: no element
	if bz == nil {
		load = k.PlayerRebuildLoad(ctx, id)
		k.PlayerSetLoad(ctx, id, load)

	} else {
		load = binary.BigEndian.Uint64(bz)
	}

	return load
}


// Sets the in-memory representation of the aggregate load for the player
func (k Keeper) PlayerSetLoad(ctx sdk.Context, id uint64, amount uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerLoadKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set(GetPlayerIDBytes(id), bz)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventPlayerLoad{Body: &types.EventBodyKeyPair{Key: id, Value: amount}})
}


// Rebuilds the current load by iterating through all player objects
func (k Keeper) PlayerRebuildLoad(ctx sdk.Context, playerId uint64) (load uint64) {

    // Add the static player draw
	load = types.PlayerPassiveDraw

    // Add all the active Structs
	structures := k.GetAllStruct(ctx)
    for _, structure := range structures {
        if (structure.Owner == playerId) {
            if (structure.Status == "ACTIVE") {
                load += structure.PassiveDraw
            }
        }
    }

	return
}


func (k Keeper) PlayerDecrementLoad(ctx sdk.Context, id uint64, amount uint64) (newLoad uint64, err error) {
	currentLoad := k.PlayerGetLoad(ctx, id)

	if amount > currentLoad {
		// this really shouldn't happen. Throw an error I guess but yeesh, this is a problem.
	} else {
		newLoad = currentLoad - amount
	}

	k.PlayerSetLoad(ctx, id, newLoad)

	return newLoad, err
}


func (k Keeper) PlayerIncrementLoad(ctx sdk.Context, player types.Player, amount uint64) (uint64, error) {

	currentLoad := k.PlayerGetLoad(ctx, player.Id)
	newLoad := currentLoad + amount

    substation, substationFound := k.GetSubstation(ctx, player.SubstationId, false)
    if (!substationFound) {
    	return 0, sdkerrors.Wrapf(types.ErrSubstationNotFound, "Player substation (substation-%d) no longer exists (that's bad)", substation.Id)
    }


	if newLoad > substation.PlayerConnectionAllocation {
		return 0, sdkerrors.Wrapf(types.ErrSubstationAvailableCapacityInsufficient, "source (substation-%d) used for allocation insufficient", substation.Id)
	}

	k.PlayerSetLoad(ctx, player.Id, newLoad)

	return newLoad, nil
}
