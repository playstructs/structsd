package keeper

import (
	"structs/x/structs/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

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


// AppendAgreement appends a agreement in the store with the ID of the related Allocation
func (cc *CurrentContext) NewAgreement(agreement types.Agreement) (*AgreementCache) {

	cc.k.SetAgreementProviderIndex(cc.ctx, agreement.ProviderId, agreement.Id)
	cc.k.SetAgreementExpirationIndex(cc.ctx, agreement.EndBlock, agreement.Id)

	cc.agreements[agreement.Id] = &AgreementCache{
            AgreementId: agreement.Id,
            CC: cc,

            Changed: true,
            Deleted: false,
            Agreement: agreement,
            AgreementLoaded: true,

            DurationRemainingLoaded: false,
            DurationPastLoaded:      false,
            DurationLoaded:          false,

            CurrentBlockLoaded: false,
        }


	return cc.agreements[agreement.Id]
}



func (cc *CurrentContext) AgreementExpirations() {
	cc.k.logger.Debug("Checking for Expired Agreements")

	uctx := sdk.UnwrapSDKContext(cc.ctx)
	currentBlock := uint64(uctx.BlockHeight())

	// Get List of Agreements
	agreements := cc.k.GetAllAgreementIdByExpirationIndex(cc.ctx, currentBlock)
	for _, agreementId := range agreements {
		cc.k.logger.Info("Expired Agreement", "agreementId", agreementId)
		agreement := cc.GetAgreement(agreementId)
		agreement.GetProvider().Checkpoint()
		agreement.Expire()
	}
}
