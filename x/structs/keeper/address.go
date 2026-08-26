package keeper

import (
	"encoding/binary"
	"context"

	"cosmossdk.io/store/prefix"
    "github.com/cosmos/cosmos-sdk/runtime"

	sdk "github.com/cosmos/cosmos-sdk/types"
	storetypes "cosmossdk.io/store/types"

	"structs/x/structs/types"
)


func (k Keeper) GetPlayerIndexFromAddress(ctx context.Context, address string) (uint64) {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressPlayerKey))

	bz := store.Get(types.KeyPrefix(address))

	// Address Not in Memory: no element
	if bz == nil  {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

// SetPlayerIndexForAddress associates an address with a player and makes sure
// the address has an auth account, so a newly registered key can sign straight
// away instead of having to receive coins first.
//
// The index row is an ordinary KV write keyed by the address string, so it is
// order-independent and is always written. Provisioning the auth account is
// not: it consumes the account keeper's global monotonic account-number
// sequence, which makes it the one write in the whole commit path whose result
// depends on when it runs. CommitAll therefore commits in sorted key order, via
// commitCaches.
//
// The parse is what decides whether an account can be provisioned at all. The
// old form discarded the error, which left the empty AccAddress, found no
// account for it, and so created and *numbered* an auth account for the empty
// address — burning a sequence number on a non-address. Provisioning is now
// skipped in that case and the error returned, meaning "associated, but this
// string cannot be a signing address". Real chains never get here:
// GenesisState.Validate rejects a malformed AddressList, and every transaction
// path supplies an address checked against PubKeyToBech32 or an already-parsed
// AccAddress.
func (k Keeper) SetPlayerIndexForAddress(ctx context.Context, address string, playerIndex uint64) error {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressPlayerKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, playerIndex)

	store.Set(types.KeyPrefix(address), bz)

    ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address, PlayerIndex: playerIndex, RegistrationStatus: types.RegistrationStatus_approved}})

	// Add the Account keeper record
	playerAccAddress, errParam := sdk.AccAddressFromBech32(address)
	if errParam != nil {
		return types.NewAddressValidationError(address, "invalid_format")
	}

	playerAuthAccount := k.accountKeeper.GetAccount(ctx, playerAccAddress)
    if playerAuthAccount == nil {
        playerAuthAccount = k.accountKeeper.NewAccountWithAddress(ctx, playerAccAddress)
        k.accountKeeper.SetAccount(ctx, playerAuthAccount)
    }

	return nil
}

// GetAddressProofNonce returns the registration-proof nonce for an address.
// An address that has never registered has no row and starts at 0.
func (k Keeper) GetAddressProofNonce(ctx context.Context, address string) uint64 {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressNonceKey))

	bz := store.Get(types.KeyPrefix(address))
	if bz == nil {
		return 0
	}

	return binary.BigEndian.Uint64(bz)
}

// SetAddressProofNonce writes the registration-proof nonce for an address.
func (k Keeper) SetAddressProofNonce(ctx context.Context, address string, nonce uint64) {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressNonceKey))

	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, nonce)

	store.Set(types.KeyPrefix(address), bz)
}

// GetAllAddressProofNonceExport returns every stored registration-proof nonce.
//
// Exported separately from GetAllAddressExport because the two stores hold
// different sets: a revoked address keeps its nonce and loses its association,
// and that is precisely the row a restored chain must not forget. Dropping it
// would reset the address to nonce 0 and make its original proof live again.
func (k Keeper) GetAllAddressProofNonceExport(ctx context.Context) (list []*types.AddressNonceRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressNonceKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, &types.AddressNonceRecord{Address: string(iterator.Key()), Nonce: binary.BigEndian.Uint64(iterator.Value())})
	}

	return
}

// RevokePlayerIndexForAddress removes the address association.
//
// It deliberately leaves the AddressNonceKey row alone. The nonce is what makes
// a registration proof single-use, and clearing it here would hand the proof
// back its validity at exactly the moment the address stops being watched.
func (k Keeper) RevokePlayerIndexForAddress(ctx context.Context, address string, playerIndex uint64)  {
    store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressPlayerKey))
	store.Delete(types.KeyPrefix(address))

    ctxSDK := sdk.UnwrapSDKContext(ctx)
	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address, PlayerIndex: playerIndex, RegistrationStatus: types.RegistrationStatus_revoked}})
}

// GetAllAddressExport returns all player addresses
func (k Keeper) GetAllAddressExport(ctx context.Context) (list []*types.AddressRecord) {
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), types.KeyPrefix(types.AddressPlayerKey))
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		list = append(list, &types.AddressRecord{Address: string(iterator.Key()), PlayerIndex: binary.BigEndian.Uint64(iterator.Value())})
	}

	return
}


func (k Keeper) AddressEmitActivity(ctx context.Context, address string) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressActivity{&types.AddressActivity{Address: address, BlockHeight: ctxSDK.BlockHeight(), BlockTime: ctxSDK.HeaderInfo().Time.UTC() }})
}