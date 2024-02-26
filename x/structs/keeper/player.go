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
	store.Set(GetObjectID(player.Id), appendedValue)

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

	store.Set(GetObjectID(player.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})
}

// GetPlayer returns a player from its id
func (k Keeper) GetPlayer(ctx sdk.Context, id uint64) (val types.Player, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	b := store.Get(GetObjectID(id))
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
	store.Delete(GetObjectID(id))
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






