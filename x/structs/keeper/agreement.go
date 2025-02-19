package keeper

import (
	//"encoding/binary"
    "context"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

	//sdkerrors "cosmossdk.io/errors"

)

// AppendAgreement appends a agreement in the store with a new id
// TODO
func (k Keeper) AppendAgreement(
	ctx context.Context,
	agreement types.Agreement,
) (err error) {

	return  nil
}

func (k Keeper) SetAgreement(ctx context.Context, agreement types.Agreement) (types.Agreement, error){

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
    b := k.cdc.MustMarshal(&agreement)
    store.Set([]byte(agreement.Id), b)

	//ctxSDK := sdk.UnwrapSDKContext(ctx)
    //_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAgreement{Agreement: &agreement})

    return agreement,  nil

}



// ImportAgreement set a specific agreement in the store
// Assumes Grid updates happen elsewhere
func (k Keeper) ImportAgreement(ctx context.Context, agreement types.Agreement){
    //k.SetAgreementSourceIndex(ctx, agreement.SourceObjectId, agreement.Id)
    //k.SetAgreementDestinationIndex(ctx, agreement.DestinationId, agreement.Id)

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
    b := k.cdc.MustMarshal(&agreement)
    store.Set([]byte(agreement.Id), b)

	//ctxSDK := sdk.UnwrapSDKContext(ctx)
    //_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAgreement{Agreement: &agreement})
}



// RemoveAgreement removes a agreement from the store
func (k Keeper) RemoveAgreement(ctx context.Context, agreementId string) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
	store.Delete([]byte(agreementId))

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: agreementId})
}

// DestroyAgreement updates grid attributes before calling RemoveAgreement
func (k Keeper) DestroyAgreement(ctx context.Context, agreementId string) (destroyed bool){
    agreement, agreementFound := k.GetAgreement(ctx, agreementId)

    _ = agreement
    if agreementFound {
    	destroyed = true
    } else {
        destroyed = false
    }

    return
}


// GetAgreement returns a agreement from its id
func (k Keeper) GetAgreement(ctx context.Context, agreementId string) (val types.Agreement, found bool) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
	b := store.Get([]byte(agreementId))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)

	return val, true
}

// GetAllAgreement returns all agreement
func (k Keeper) GetAllAgreement(ctx context.Context) (list []types.Agreement) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Agreement
		k.cdc.MustUnmarshal(iterator.Value(), &val)

		list = append(list, val)
	}

	return
}


