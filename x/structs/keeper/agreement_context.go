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


func (cc *CurrentContext) GenesisImportAgreement(agreement types.Agreement) {
	cache := cc.GetAgreement(agreement.Id)
	cache.Agreement = agreement
	cache.AgreementLoaded = true
	cache.Changed = true
	cache.EndBlockChanged = true

	cc.k.SetAgreementProviderIndex(cc.ctx, agreement.ProviderId, agreement.Id)

	provider := cc.GetProvider(agreement.ProviderId)
	cc.SetGridAttributeIncrement(provider.AgreementLoadAttributeId, agreement.Capacity)
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



/* RebaseAgreementsForZeroHeightGenesis rewrites every agreement so its window
 * carries only the time it has left, measured from block zero.
 *
 * An agreement's StartBlock and EndBlock are absolute heights on the record
 * itself, so a genesis export carries them through unchanged. That is right for
 * a height-preserving restart and wrong for a zero-height one: an agreement
 * exported at height H with E-H blocks left is re-indexed to expire at E on a
 * chain that restarts near 1, so it supplies capacity for roughly the whole age
 * of the old chain without anybody having posted collateral for it.
 *
 * Providers are checkpointed first, so what the old window earned is settled
 * against the old clock before that clock is thrown away, and their checkpoints
 * are then rebased to zero alongside the agreements - the two have to move
 * together or Checkpoint() bills a span the agreements no longer claim.
 *
 * An agreement with nothing left is given one block rather than zero. Zero would
 * index it at a height the chain never reaches, and an expiry gets exactly one
 * attempt: it would hold capacity in the provider's load forever, which is what
 * agreement-expiry-liveness exists to catch.
 */
func (cc *CurrentContext) RebaseAgreementsForZeroHeightGenesis() error {
	uctx := sdk.UnwrapSDKContext(cc.ctx)
	exportHeight := uint64(uctx.BlockHeight())

	// Settle every provider against the old clock before it is discarded.
	// Checkpoint bills aggregate load from the checkpoint block, so it has to
	// happen while that block still means something.
	for _, provider := range cc.k.GetAllProvider(cc.ctx) {
		providerCache := cc.GetProvider(provider.Id)
		if err := providerCache.Checkpoint(); err != nil {
			return err
		}
	}

	for _, agreement := range cc.k.GetAllAgreement(cc.ctx) {
		remaining := uint64(1)
		if agreement.EndBlock > exportHeight {
			remaining = agreement.EndBlock - exportHeight
		}

		cache := cc.GetAgreement(agreement.Id)
		if !cache.LoadAgreement() {
			return types.NewObjectNotFoundError("agreement", agreement.Id)
		}

		// SetEndBlock records PreviousEndBlock, so Commit clears the row at the
		// old absolute height and writes the rebased one.
		cache.SetStartBlock(0)
		cache.SetEndBlock(remaining)
	}

	// The checkpoint clock restarts with the windows it bills against. Done
	// after the loop so a provider carrying several agreements is rebased once.
	for _, provider := range cc.k.GetAllProvider(cc.ctx) {
		cc.GetProvider(provider.Id).SetCheckpointBlock(0)
	}

	return nil
}

func (cc *CurrentContext) AgreementExpirations() {
	cc.k.logger.Debug("Checking for Expired Agreements")

	uctx := sdk.UnwrapSDKContext(cc.ctx)
	currentBlock := uint64(uctx.BlockHeight())

	// Get List of Agreements
	// This runs in the EndBlocker, so a failure on one agreement is logged and
	// the rest still expire: aborting would strand every later agreement in the
	// list with its provider load still counted, and there is no retry.
	agreements := cc.k.GetAllAgreementIdByExpirationIndex(cc.ctx, currentBlock)
	for _, agreementId := range agreements {
		cc.k.logger.Info("Expired Agreement", "agreementId", agreementId)
		agreement := cc.GetAgreement(agreementId)
		if err := agreement.Expire(); err != nil {
			cc.k.logger.Error("Expired Agreement could not be settled", "agreementId", agreementId, "error", err)
		}
	}
}
