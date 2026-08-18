package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	crypto "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	"structs/x/structs/types"
)

/* CharterAnchor resolves the height the charter difficulty is measured from.
 * An unset anchor falls back to the current height so it cannot make a fresh
 * puzzle inherit the chain's entire age.
 */
func (k Keeper) CharterAnchor(ctx context.Context) uint64 {
	if anchor, found := k.GetGuildCharterAnchor(ctx); found {
		return anchor
	}

	return uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
}

/* CharterAge is how many blocks of decay the current puzzle has accumulated.
 * It clamps at zero so an anchor ahead of the current height cannot underflow
 * into a maximally easy puzzle.
 */
func (k Keeper) CharterAge(ctx context.Context) uint64 {
	blockHeight := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	anchor := k.CharterAnchor(ctx)

	if blockHeight <= anchor {
		return 0
	}

	return blockHeight - anchor
}

// CharterDifficulty is the leading-zero requirement a charter proof must meet
// right now.
func (k Keeper) CharterDifficulty(ctx context.Context) int {
	return types.CalculateDifficulty(float64(k.CharterAge(ctx)), k.GetParams(ctx).CharterDifficultyRange())
}

/* verifyGuildCharterConsent checks a founder's offline signature authorising
 * somebody else to found their guild.
 *
 * The consenting address is a subject, not the transaction identity. It must
 * still belong to the founder and hold PermPlay when the consent is used.
 */
func (k Keeper) verifyGuildCharterConsent(cc *CurrentContext, msg *types.MsgGuildCreate, solver *PlayerCache, founder *PlayerCache, anchor uint64) error {
	if msg.Address == "" || msg.ProofPubKey == "" || msg.ProofSignature == "" {
		return types.NewPermissionError("player", solver.GetPlayerId(), "player", founder.GetPlayerId(),
			uint64(types.PermPlay), "guild_charter_consent")
	}

	decodedProofPubKey, decodeErr := hex.DecodeString(msg.ProofPubKey)
	if decodeErr != nil {
		return types.NewAddressValidationError(msg.Address, "pubkey_invalid")
	}

	if types.PubKeyToBech32(decodedProofPubKey) != msg.Address {
		return types.NewAddressValidationError(msg.Address, "proof_mismatch").
			WithPlayers(types.PubKeyToBech32(decodedProofPubKey), msg.Address)
	}

	decodedProofSignature, decodeErr := hex.DecodeString(msg.ProofSignature)
	if decodeErr != nil {
		return types.NewAddressValidationError(msg.Address, "signature_invalid")
	}

	// Some signing systems append a recovery byte that breaks verification.
	if len(decodedProofSignature) < 64 {
		return types.NewAddressValidationError(msg.Address, "signature_invalid")
	}

	pubKey := crypto.PubKey{Key: decodedProofPubKey}
	consentInput := types.GuildCharterConsentInput(sdk.UnwrapSDKContext(cc.ctx).ChainID(), founder.GetPlayerId(), msg.ReactorId, msg.EntrySubstationId, msg.Endpoint, anchor)

	if !pubKey.VerifySignature([]byte(consentInput), decodedProofSignature[:64]) {
		return types.NewAddressValidationError(msg.Address, "signature_invalid")
	}

	consentingPlayer, consentingPlayerErr := cc.GetPlayerByAddress(msg.Address)
	if consentingPlayerErr != nil {
		return consentingPlayerErr
	}

	if consentingPlayer.GetPlayerId() != founder.GetPlayerId() {
		return types.NewAddressValidationError(msg.Address, "wrong_player").
			WithPlayers(founder.GetPlayerId(), consentingPlayer.GetPlayerId())
	}

	if !cc.PermissionHasAll(GetAddressPermissionIDBytes(msg.Address), types.PermPlay) {
		return types.NewPermissionError("address", msg.Address, "player", founder.GetPlayerId(),
			uint64(types.PermPlay), "guild_charter_consent")
	}

	return nil
}

/* guildCharterReactorLive requires an existing, unjailed validator. Bonded
 * status is deliberately not required on the proof path because a guild may
 * form while its validator is still entering the active set.
 */
func (k Keeper) guildCharterReactorLive(ctx context.Context, reactor *ReactorCache) error {
	validatorAddress, addressErr := sdk.ValAddressFromBech32(reactor.GetReactor().Validator)
	if addressErr != nil {
		return types.NewObjectNotFoundError("validator", reactor.GetReactor().Validator)
	}

	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	if validatorErr != nil {
		return types.NewObjectNotFoundError("validator", validatorAddress.String())
	}

	if validator.IsJailed() {
		return types.NewReactorError("guild_charter", "validator_jailed").
			WithReactor(reactor.GetReactorId()).
			WithAddress(validatorAddress.String(), "validator")
	}

	return nil
}

/* guildCharterReactorEligible reports whether a reactor may found a guild
 * without a proof, and why not when it may not.
 *
 * The entitlement path requires a currently bonded validator so throwaway
 * validators cannot create an unbounded supply of free guilds.
 */
func (k Keeper) guildCharterReactorEligible(ctx context.Context, reactor *ReactorCache, founder *PlayerCache) error {
	blockHeight := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	eligibleHeight := reactor.GetReactor().GuildCharterEligibleHeight

	// An unstamped reactor is one the grandfathering migration missed, which
	// means it postdates the upgrade and AppendReactor should have stamped it.
	// Refusing is the safe reading; zero would otherwise mean eligible at once.
	if eligibleHeight == 0 || blockHeight < eligibleHeight {
		return types.NewReactorError("guild_charter", "not_yet_eligible").
			WithReactor(reactor.GetReactorId())
	}

	// The entitlement is one per reactor, and GuildId is the marker. It is
	// stamped on success, which is what spends it.
	if reactor.GetReactor().GuildId != "" {
		return types.NewReactorError("guild_charter", "entitlement_spent").
			WithReactor(reactor.GetReactorId())
	}

	validatorAddress, addressErr := sdk.ValAddressFromBech32(reactor.GetReactor().Validator)
	if addressErr != nil {
		return types.NewObjectNotFoundError("validator", reactor.GetReactor().Validator)
	}

	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	if validatorErr != nil {
		return types.NewObjectNotFoundError("validator", validatorAddress.String())
	}

	if validator.IsJailed() || !validator.IsBonded() {
		return types.NewReactorError("guild_charter", "validator_inactive").
			WithReactor(reactor.GetReactorId()).
			WithAddress(validatorAddress.String(), "validator")
	}

	return reactor.CanCreateGuildBy(founder)
}

/* guildCharterLeavable returns the guild a founder must leave. An owner cannot
 * leave through creation because that would strand their existing guild.
 */
func guildCharterLeavable(cc *CurrentContext, founder *PlayerCache) (*GuildCache, error) {
	if founder.GetGuildId() == "" {
		return nil, nil
	}

	oldGuild := cc.GetGuild(founder.GetGuildId())
	if oldGuild.CheckGuild() != nil {
		return nil, nil
	}

	if oldGuild.GetOwnerId() == founder.GetPlayerId() {
		return nil, types.NewGuildMembershipError(oldGuild.GetGuildId(), founder.GetPlayerId(), "is_owner")
	}

	return oldGuild, nil
}

// guildCharterLeaveAndJoin preserves an unrelated substation connection while
// moving the founder between guilds.
func guildCharterLeaveAndJoin(cc *CurrentContext, founder *PlayerCache, oldGuild *GuildCache, guildId string, entrySubstationId string) {
	if oldGuild != nil {
		if founder.GetSubstationId() != "" && founder.GetSubstationId() == oldGuild.GetEntrySubstationId() {
			founder.DisconnectSubstation()
		}
	}

	founder.SetGuild(guildId)
	founder.SetGuildRank(1)

	if entrySubstationId != "" && founder.GetSubstationId() == "" {
		founder.MigrateSubstation(entrySubstationId)
	}
}
