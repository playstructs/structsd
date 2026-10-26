package keeper

import (
	"encoding/binary"
    "context"

    "github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"structs/x/structs/types"
    "strconv"


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


/* AgreementExpirationHeightHasRoomFor reports whether agreementId may take (or
 * keep) a slot in the expiration bucket for block.
 *
 * Counting rather than maintaining a counter is deliberate. The iteration stops
 * at the cap, so this is at most AgreementExpirationBucketCap+1 reads however
 * large the bucket actually is - which matters, because a chain upgrading into
 * this rule may already hold buckets far past the cap and there is no counter to
 * migrate, no genesis row to rebuild, and nothing to drift out of step with the
 * index it is supposed to describe. An over-full bucket simply accepts nothing
 * new until it drains.
 *
 * An agreement already indexed at this height is always allowed: a capacity
 * change that happens to land back on the same block is not adding work, and
 * refusing it would make a re-price fail for no reason.
 */
func (k Keeper) AgreementExpirationHeightHasRoomFor(ctx context.Context, block uint64, agreementId string) bool {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementExpirationKeyPrefix(block))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		if string(iterator.Key()) == agreementId {
			return true
		}

		count++
		if count >= types.AgreementExpirationBucketCap {
			return false
		}
	}

	return true
}

func (k Keeper) SetAgreementExpirationIndex(ctx context.Context, block uint64, agreementId string) (err error) {
    providerIndexStore := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), AgreementExpirationKeyPrefix(block))

    k.logger.Info("New Agreement ", "agreementId", agreementId, "expirationHeight", block)

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




