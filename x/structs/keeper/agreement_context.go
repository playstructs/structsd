package keeper

// GetAgreement returns an AgreementCache by ID, loading from store if not already cached.
func (cc *CurrentContext) GetAgreement(agreementId string) *AgreementCache {
	if cache, exists := cc.agreements[agreementId]; exists {
		return cache
	}

	cc.agreements[agreementId] = &AgreementCache{
            AgreementId: agreementId,
            CC: cc,

            Changed: false,
            Deleted: false,
            AgreementLoaded: false,

            DurationRemainingLoaded: false,
            DurationPastLoaded:      false,
            DurationLoaded:          false,

            CurrentBlockLoaded: false,
        }

	return cc.agreements[agreementId]
}

// RegisterAgreement registers an externally created AgreementCache with the context.
func (cc *CurrentContext) RegisterAgreement(cache *AgreementCache) {
	if cache == nil {
		return
	}
	cache.CC = cc
	cc.agreements[cache.AgreementId] = cache
}

