package keeper

import (
	"structs/x/structs/types"

    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetProvider returns a ProviderCache by ID, loading from store if not already cached.
// Returns nil if the provider has been deleted in this context.
func (cc *CurrentContext) GetProvider(providerId string) *ProviderCache {
	if cache, exists := cc.providers[providerId]; exists {
		return cache
	}

	cc.providers[providerId] = &ProviderCache{
            ProviderId: providerId,
            CC: cc,

            Changed: false,
            ProviderLoaded:  false,

            CheckpointBlockAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, providerId),
            AgreementLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, providerId),
        }

	return cc.providers[providerId]
}

/* GetExistingProvider resolves a provider id a message chose.
 *
 * GetProvider is a cache allocator: it wraps any string, reads nothing, and
 * reports no miss. That is right for an id taken off something already loaded
 * from state - an agreement's provider, a pool index row - and wrong for one a
 * transaction supplied.
 *
 * The permission check is not a backstop, for the same reason it was not one in
 * ProviderCreate and AllocationTransfer: object permissions are keyed by the raw
 * id string with no type namespacing, and registration grants every player
 * PermAll on their own player id. Submitting that id as a provider id therefore
 * collides with a record that genuinely exists, and CanBeUpdatedBy passes on a
 * provider that does not.
 *
 * What follows is a phantom cache, and the thing to know about a phantom is that
 * it behaves: it reads as a zero-valued provider rather than failing, so the
 * handler mutates it and commits. Today that commit happens to panic in the KV
 * store on the empty Provider.Id, and BaseApp turns the panic into a failed
 * transaction - but a refusal that only happens because an unrelated write
 * panics is not a check. The same shape wrote real state in AllocationTransfer,
 * where the write was keyed by something non-empty and nothing tripped.
 */
func (cc *CurrentContext) GetExistingProvider(providerId string) (*ProviderCache, error) {
	if !ObjectIdHasType(providerId, types.ObjectType_provider) {
		return nil, types.NewObjectNotFoundError("provider", providerId)
	}

	provider := cc.GetProvider(providerId)
	if err := provider.CheckProvider(); err != nil {
		return nil, err
	}

	return provider, nil
}

func (cc *CurrentContext) GenesisImportProvider(provider types.Provider) {
	cache := cc.GetProvider(provider.Id)
	cache.Provider = provider
	cache.ProviderLoaded = true
	cache.Changed = true
}

// AppendProvider appends a provider in the store with a new id
func (cc *CurrentContext) NewProvider(provider types.Provider) (*ProviderCache) {

	// Define the provider id
	provider.Index = cc.k.GetProviderCount(cc.ctx)
    cc.k.SetProviderCount(cc.ctx, provider.Index+1)
	// Set the ID of the appended value
	providerId := GetObjectID(types.ObjectType_provider, provider.Index)
	provider.Id = providerId

	cc.providers[providerId] = &ProviderCache{
            ProviderId: providerId,
            CC: cc,

            Changed: true,
            Provider: provider,
            ProviderLoaded: true,

            CheckpointBlockAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, providerId),
            AgreementLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, providerId),
        }

	ctxSDK := sdk.UnwrapSDKContext(cc.ctx)

	// Set the Checkpoint to current block
	cc.SetGridAttribute(cc.providers[providerId].CheckpointBlockAttributeId, uint64(ctxSDK.BlockHeight()))

	// Create the Collateral Pool
	providerCollateralDetails := types.ProviderCollateralPool+provider.Id
	providerCollateralAddress := authtypes.NewModuleAddress(providerCollateralDetails)
	providerCollateralAccount := cc.k.accountKeeper.NewAccountWithAddress(cc.ctx, providerCollateralAddress)
	cc.k.accountKeeper.SetAccount(cc.ctx, providerCollateralAccount)

	// Create the Earnings Pool
	providerEarningsDetails := types.ProviderEarningsPool+provider.Id
	providerEarningsAddress := authtypes.NewModuleAddress(providerEarningsDetails)
	providerEarningsAccount := cc.k.accountKeeper.NewAccountWithAddress(cc.ctx, providerEarningsAddress)
	cc.k.accountKeeper.SetAccount(cc.ctx, providerEarningsAccount)

	providerCollateralAddressStr := providerCollateralAddress.String()
	providerEarningsAddressStr := providerEarningsAddress.String()
	cc.k.IndexProviderPoolAddresses(cc.ctx, provider.Id)

	cc.k.logger.Info("Provider Created",
		"providerId", provider.Id,
		"collateralPoolDetails", providerCollateralDetails,
		"collateralPoolAddress", providerCollateralAddressStr,
		"earningsPoolDetails", providerEarningsDetails,
		"earningsPoolAddress", providerEarningsAddressStr)

	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProviderAddress{&types.EventProviderAddressDetail{ProviderId: provider.Id, CollateralPool: providerCollateralAddressStr, EarningPool: providerEarningsAddressStr}})

	return cc.providers[providerId]
}