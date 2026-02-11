package keeper

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

// TODO HERE
// AppendProvider appends a provider in the store with a new id
func (cc *CurrentContext) NewProvider(provider types.Provider) (*ProviderCache) {

	// Define the provider id
	provider.Index := cc.k.GetProviderCount(cc.ctx)
    k.SetProviderCount(ctx, count+1)
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

	cc.k.logger.Info("Provider Created",
		"providerId", provider.Id,
		"collateralPoolDetails", providerCollateralDetails,
		"collateralPoolAddress", providerCollateralAddressStr,
		"earningsPoolDetails", providerEarningsDetails,
		"earningsPoolAddress", providerEarningsAddressStr)

	_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProviderAddress{&types.EventProviderAddressDetail{ProviderId: provider.Id, CollateralPool: providerCollateralAddressStr, EarningPool: providerEarningsAddressStr}})

	return cc.providers[providerId]
}