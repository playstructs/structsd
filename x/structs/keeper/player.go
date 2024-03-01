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
) types.Player {
	// Create the player
	count := k.GetPlayerCount(ctx)

	// Set the ID of the appended value
	player.Id = GetObjectID(types.ObjectType_player, count)
	player.Index = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	appendedValue := k.cdc.MustMarshal(&player)
	store.Set([]byte(player.Id), appendedValue)

	// Update player count
	k.SetPlayerCount(ctx, count+1)

	//Add Address records
	k.SetPlayerIndexForAddress(ctx, player.Creator, player.Index)

	addressPermissionId := GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.Permission(types.AddressPermissionAll))

	// Add the Account keeper record
	// This is needed for the proxy account creation
	playerAccAddress, _ := sdk.AccAddressFromBech32(player.Creator)
	playerAuthAccount := k.accountKeeper.GetAccount(ctx, playerAccAddress)
    if playerAuthAccount == nil {
        playerAuthAccount = k.accountKeeper.NewAccountWithAddress(ctx, playerAccAddress)
        k.accountKeeper.SetAccount(ctx, playerAuthAccount)
    }

    // Add the initial Player Load
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, player.Id), types.PlayerPassiveDraw)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})

	return player
}

// SetPlayer set a specific player in the store
func (k Keeper) SetPlayer(ctx sdk.Context, player types.Player) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	b := k.cdc.MustMarshal(&player)

	store.Set([]byte(player.Id), b)

	_ = ctx.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})
}

// GetPlayer returns a player from its id
func (k Keeper) GetPlayer(ctx sdk.Context, playerId string, full bool) (val types.Player, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	b := store.Get([]byte(playerId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

    if (full) {
        val.Load      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
        val.Capacity  = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))

        val.StructsLoad           = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, val.Id))
        val.CapacitySecondary    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, val.SubstationId))

    	playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
    	val.Storage   = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")
    }

	return val, true
}

func (k Keeper) GetPlayerFromIndex(ctx sdk.Context, playerIndex uint64, full bool) (val types.Player, found bool) {
    val, found = k.GetPlayer(ctx, GetObjectID(types.ObjectType_player, playerIndex), full)
    return
}

// RemovePlayer removes a player from the store
func (k Keeper) RemovePlayer(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	store.Delete([]byte(id))
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayer(ctx sdk.Context, full bool) (list []types.Player) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)


        if (full) {
            val.Load      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
            val.Capacity  = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))

            val.StructsLoad           = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, val.Id))
            val.CapacitySecondary    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, val.SubstationId))

            playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
            val.Storage   = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")
        }

		list = append(list, val)
	}

	return
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayerBySubstation(ctx sdk.Context, substationId string, full bool) (list []types.Player) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PlayerKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if (val.SubstationId == substationId) {

            if (full) {
                val.Load      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, val.Id))
                val.Capacity  = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, val.Id))

                val.StructsLoad           = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, val.Id))
                val.CapacitySecondary    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, val.SubstationId))


                playerAcc, _ := sdk.AccAddressFromBech32(val.PrimaryAddress)
                val.Storage   = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")
            }

            list = append(list, val)
		}
	}

	return
}

// Technically more of an InGet than an UpSert
func (k Keeper) UpsertPlayer(ctx sdk.Context, playerAddress string, full bool) (player types.Player) {
    playerIndex := k.GetPlayerIndexFromAddress(ctx, playerAddress)

    if (playerIndex == 0) {
        // No Player Found, Creating..
        player.Creator = playerAddress
        player.PrimaryAddress = playerAddress

        player = k.AppendPlayer(ctx, player)

    } else {
        player, _ = k.GetPlayerFromIndex(ctx, playerIndex, full)
    }

    return player
}






