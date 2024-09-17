package keeper

import (
	"encoding/binary"
	"context"
	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"

	"fmt"
)



// GetPlanetAttributeID returns the string representation of the ID
func GetPlanetAttributeID(planetAttributeType types.PlanetAttributeType, objectType types.ObjectType, objectId uint64) string {
    id := fmt.Sprintf("%d-%d-%d", planetAttributeType, objectType, objectId)
	return id
}

// GetPlanetAttributeIDByObjectId returns the string representation of the ID
func GetPlanetAttributeIDByObjectId(planetAttributeType types.PlanetAttributeType, objectId string) string {
    id := fmt.Sprintf("%d-%s", planetAttributeType, objectId)
	return id
}


func (k Keeper) GetPlanetAttribute(ctx context.Context, planetAttributeId string) (amount uint64) {
	planetAttributeStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlanetAttributeKey))

	bz := planetAttributeStore.Get([]byte(planetAttributeId))

	if bz == nil {
        // return error?
        // err =
		amount = 0
	} else {
		amount = binary.BigEndian.Uint64(bz)
	}

	return
}

func (k Keeper) ClearPlanetAttribute(ctx context.Context, planetAttributeId string) () {
	planetAttributeStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlanetAttributeKey))
	planetAttributeStore.Delete([]byte(planetAttributeId))
}


func (k Keeper) SetPlanetAttribute(ctx context.Context, planetAttributeId string, amount uint64) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.PlanetAttributeKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, amount)

	store.Set([]byte(planetAttributeId), bz)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlanetAttribute{&types.PlanetAttributeRecord{AttributeId: planetAttributeId, Value: amount}})
    fmt.Printf("Planet Change (Set): (%s) %d \n", planetAttributeId, amount)
}

func (k Keeper) SetPlanetAttributeDelta(ctx context.Context, planetAttributeId string, oldAmount uint64, newAmount uint64) (amount uint64, err error) {
    currentAmount := k.GetPlanetAttribute(ctx, planetAttributeId)

    var resetAmount uint64
    if (oldAmount < currentAmount) {
        resetAmount = currentAmount - oldAmount
    }

    amount = resetAmount + newAmount

    fmt.Printf("Planet Change (Delta): (%s) %d to %d \n", planetAttributeId, oldAmount, newAmount)
    k.SetPlanetAttribute(ctx, planetAttributeId, amount)

    return
}

func (k Keeper) SetPlanetAttributeDecrement(ctx context.Context, planetAttributeId string, decrementAmount uint64) (amount uint64, err error) {
    currentAmount := k.GetPlanetAttribute(ctx, planetAttributeId)

    if (decrementAmount < currentAmount) {
        amount = currentAmount - decrementAmount
    }

    fmt.Printf("Planet Change (Decrement): (%s) %d \n", planetAttributeId, decrementAmount)
    k.SetPlanetAttribute(ctx, planetAttributeId, amount)

    return
}

func (k Keeper) SetPlanetAttributeIncrement(ctx context.Context, planetAttributeId string, incrementAmount uint64) (amount uint64) {
    currentAmount := k.GetPlanetAttribute(ctx, planetAttributeId)

    amount = currentAmount + incrementAmount

    fmt.Printf("Planet Change (Increment): (%s) %d \n", planetAttributeId, incrementAmount)
    k.SetPlanetAttribute(ctx, planetAttributeId, amount)

    return
}



func (k Keeper) GetPlanetAttributesByObject(ctx context.Context, objectId string) (types.PlanetAttributes) {
    return types.PlanetAttributes{
        PlanetaryShield:                        k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_planetaryShield, objectId)),
        RepairNetworkQuantity:                  k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_repairNetworkQuantity, objectId)),
        DefensiveCannonQuantity:                k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_defensiveCannonQuantity, objectId)),
        CoordinatedGlobalShieldNetworkQuantity: k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_coordinatedGlobalShieldNetworkQuantity, objectId)),

        LowOrbitBallisticsInterceptorNetworkQuantity:           k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkQuantity, objectId)),
        AdvancedLowOrbitBallisticsInterceptorNetworkQuantity:   k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedLowOrbitBallisticsInterceptorNetworkQuantity, objectId)),

        LowOrbitBallisticsInterceptorNetworkSuccessRateNumerator:   k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateNumerator, objectId)),
        LowOrbitBallisticsInterceptorNetworkSuccessRateDenominator: k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_lowOrbitBallisticsInterceptorNetworkSuccessRateDenominator, objectId)),

        OrbitalJammingStationQuantity:          k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_orbitalJammingStationQuantity, objectId)),
        AdvancedOrbitalJammingStationQuantity:  k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_advancedOrbitalJammingStationQuantity, objectId)),

        BlockStartRaid:                         k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_blockStartRaid, objectId)),
  }
}