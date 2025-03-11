package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"

    storetypes "cosmossdk.io/store/types"

	"context"
    //"math"
    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)


// GetPlayerCount get the total number of player
func (k Keeper) GetPlayerCount(ctx context.Context) uint64 {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
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
func (k Keeper) SetPlayerCount(ctx context.Context, count uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte{})
	byteKey := types.KeyPrefix(types.PlayerCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendPlayer appends a player in the store with a new id and update the count
func (k Keeper) AppendPlayer(
	ctx context.Context,
	player types.Player,
) types.Player {
	ctxSDK := sdk.UnwrapSDKContext(ctx)

	// Create the player
	count := k.GetPlayerCount(ctx)
    player.Index = count

	// Set the ID of the appended value
	player.Id = GetObjectID(types.ObjectType_player, player.Index)

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	appendedValue := k.cdc.MustMarshal(&player)
	store.Set([]byte(player.Id), appendedValue)

	// Update player count
	k.SetPlayerCount(ctx, player.Index + 1)

	//Add Address records
	k.SetPlayerIndexForAddress(ctx, player.Creator, player.Index)

	addressPermissionId := GetAddressPermissionIDBytes(player.Creator)
	k.PermissionAdd(ctx, addressPermissionId, types.PermissionAll)

    // Add the initial Player Load
    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), types.PlayerPassiveDraw)

    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})

	return player
}

// SetPlayer set a specific player in the store
func (k Keeper) SetPlayer(ctx context.Context, player types.Player) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	b := k.cdc.MustMarshal(&player)

	store.Set([]byte(player.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})
}

// GetPlayer returns a player from its id
func (k Keeper) GetPlayer(ctx context.Context, playerId string) (val types.Player, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	b := store.Get([]byte(playerId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

func (k Keeper) GetPlayerFromIndex(ctx context.Context, playerIndex uint64) (val types.Player, found bool) {
    val, found = k.GetPlayer(ctx, GetObjectID(types.ObjectType_player, playerIndex))
    return
}

// RemovePlayer removes a player from the store
func (k Keeper) RemovePlayer(ctx context.Context, playerId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	store.Delete([]byte(playerId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ ObjectId: playerId })
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayer(ctx context.Context) (list []types.Player) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}

// GetAllPlayer returns all player
func (k Keeper) GetAllPlayerBySubstation(ctx context.Context, substationId string) (list []types.Player) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlayerKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Player
		k.cdc.MustUnmarshal(iterator.Value(), &val)

        if (val.SubstationId == substationId) {
            list = append(list, val)
		}
	}

	return
}

// Technically more of an InGet than an UpSert
func (k Keeper) UpsertPlayer(ctx context.Context, playerAddress string) (player types.Player) {
    playerIndex := k.GetPlayerIndexFromAddress(ctx, playerAddress)

    if (playerIndex == 0) {
        // No Player Found, Creating..
        player.Creator = playerAddress
        player.PrimaryAddress = playerAddress

        player = k.AppendPlayer(ctx, player)

    } else {
        player, _ = k.GetPlayerFromIndex(ctx, playerIndex)
    }

    return player
}


/*
 The old charge function that actually used the math.

 Now we handle the math externally and just calculate based on
func (k Keeper) GetPlayerCharge(ctx context.Context, playerId string) (charge uint64) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)

    // Volts*(1-power(exp(1),-(BlockSpan/(Resistance*Capacitance))))
    // Volts = 100000000
    // Resistance = 100
    // Capacitor (capacitance) = 10

    lastActionBlock := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId))
    blockSpan := uint64(ctxSDK.BlockHeight()) - lastActionBlock

    // TODO / FIX
    // NGL, these floats freak me out a bit. Not sure if they'll be a source of consensus failures down the road
    result := types.Charge_Volts * (1 - math.Pow(math.Exp(1), -(float64(blockSpan)/(types.Charge_Resistance*types.Charge_Capacitance))))
    charge = uint64(result)

	return
}
*/

func (k Keeper) GetPlayerCharge(ctx context.Context, playerId string) (charge uint64) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)

    lastActionBlock := k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId))
    charge = uint64(ctxSDK.BlockHeight()) - lastActionBlock

	return
}

func (k Keeper) DischargePlayer(ctx context.Context, playerId string) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)
    k.SetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, playerId), uint64(ctxSDK.BlockHeight()))
}


/*
    message PlayerInventory {
      cosmos.base.v1beta1.Coin rocks = 13
      [(gogoproto.nullable) = false, (gogoproto.castrepeated) = "github.com/cosmos/cosmos-sdk/types.Coins"];
    }
*/

func (k Keeper) GetPlayerInventory(ctx context.Context, primaryAddress string) (types.PlayerInventory){

     playerAcc, _ := sdk.AccAddressFromBech32(primaryAddress)
     storage := k.bankKeeper.SpendableCoin(ctx, playerAcc, "ualpha")

     return types.PlayerInventory{Rocks: storage}

}