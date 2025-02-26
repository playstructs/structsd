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

// AppendAgreement appends a agreement in the store with the ID of the related Allocation
func (k Keeper) AppendAgreement(
	ctx context.Context,
	agreement types.Agreement,
) (err error) {
    k.SetAgreementProviderIndex(ctx, agreement.ProviderId, agreement.Id)

	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
	appendedValue := k.cdc.MustMarshal(&agreement)
	store.Set([]byte(agreement.Id), appendedValue)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAgreement{Agreement: &agreement})

	return nil
}

func (k Keeper) SetAgreement(ctx context.Context, agreement types.Agreement) (types.Agreement, error){

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
    b := k.cdc.MustMarshal(&agreement)
    store.Set([]byte(agreement.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAgreement{Agreement: &agreement})

    return agreement,  nil
}



// ImportAgreement set a specific agreement in the store
// Assumes Grid updates happen elsewhere
func (k Keeper) ImportAgreement(ctx context.Context, agreement types.Agreement){
    k.SetAgreementProviderIndex(ctx, agreement.ProviderId, agreement.Id)

    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
    b := k.cdc.MustMarshal(&agreement)
    store.Set([]byte(agreement.Id), b)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAgreement{Agreement: &agreement})
}



// RemoveAgreement removes a agreement from the store
func (k Keeper) RemoveAgreement(ctx context.Context, agreement types.Agreement) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AgreementKey))
	store.Delete([]byte(agreement.Id))

	 k.RemoveAgreementProviderIndex(ctx, agreement.ProviderId, agreement.Id)

	ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventDelete{ObjectId: agreement.Id})
}

func (k Keeper) AgreementExpirations(ctx context.Context) {

    uctx := sdk.UnwrapSDKContext(ctx)
    currentBlock := uint64(uctx.BlockHeight())

    // Get List of Agreements
    agreements := k.GetAllAgreementIdByExpirationIndex(ctx, currentBlock)
    for _, agreementId := range agreements {
        agreement := k.GetAgreementCacheFromId(ctx, agreementId)
        agreement.GetProvider().Checkpoint()
        agreement.Expire()
    }

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


