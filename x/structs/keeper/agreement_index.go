package keeper

import (
	"encoding/binary"
    "context"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"structs/x/structs/types"
    "strconv"
    "fmt"

)

func AgreementProviderKeyPrefix(providerId string) []byte {
	return []byte(types.AgreementProviderKey + providerId + "/")
}

func AgreementExpirationKeyPrefix(block uint64) []byte {
	return []byte(types.AgreementExpirationKey + strconv.FormatUint(block, 10) + "/")
}

func (k Keeper) SetAgreementProviderIndex(ctx context.Context, providerId string, agreementId string) (err error) {
    providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementProviderKeyPrefix(providerId))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	providerIndexStore.Set([]byte(agreementId), bz)

	return err
}

func (k Keeper) RemoveAgreementProviderIndex(ctx context.Context, providerId string, agreementId string) (err error) {
    providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementProviderKeyPrefix(providerId))
	providerIndexStore.Delete([]byte(agreementId))

	return err
}


func (k Keeper) GetAllAgreementIdByProviderIndex(ctx context.Context, providerId string) (list []string) {
	providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementProviderKeyPrefix(providerId))
	iterator := storetypes.KVStorePrefixIterator(providerIndexStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Key()))
	}

    return
}

func (k Keeper) GetAllAgreementByProviderIndex(ctx context.Context, providerId string) (list []types.Agreement) {
	providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementProviderKeyPrefix(providerId))
	iterator := storetypes.KVStorePrefixIterator(providerIndexStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		val, found := k.GetAgreement(ctx, string(iterator.Key()))
		if found {
		    list = append(list, val)
    	}
    }
    return
}



func (k Keeper) SetAgreementExpirationIndex(ctx context.Context, block uint64, agreementId string) (err error) {
    providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementExpirationKeyPrefix(block))

    fmt.Printf("New Agreement %s will expire on %d \n", agreementId, block)

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, 1)

	providerIndexStore.Set([]byte(agreementId), bz)

	return err
}

func (k Keeper) RemoveAgreementExpirationIndex(ctx context.Context, block uint64, agreementId string) (err error) {
    providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementExpirationKeyPrefix(block))
	providerIndexStore.Delete([]byte(agreementId))

	return err
}


func (k Keeper) GetAllAgreementIdByExpirationIndex(ctx context.Context, block uint64) (list []string) {
	providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementExpirationKeyPrefix(block))
	iterator := storetypes.KVStorePrefixIterator(providerIndexStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Key()))
	}

    return
}

func (k Keeper) GetAllAgreementByExpirationIndex(ctx context.Context, block uint64) (list []types.Agreement) {
	providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementExpirationKeyPrefix(block))
	iterator := storetypes.KVStorePrefixIterator(providerIndexStore, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		val, found := k.GetAgreement(ctx, string(iterator.Key()))
		if found {
		    list = append(list, val)
    	}
    }
    return
}




