package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	crypto "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	"structs/x/structs/types"
)

/* CharterAnchor resolves the height the charter difficulty is measured from.
 *
 * Falls back to the current height when no anchor has ever been stamped, which
 * is the safe direction: an unset anchor read as zero would make the age the
 * whole chain's history and hand out a guild for one leading zero. Genesis
 * import and the upgrade handler both stamp it, so this fallback should only be
 * reachable from a test keeper.
 */
func (k Keeper) CharterAnchor(ctx context.Context) uint64 {
	if anchor, found := k.GetGuildCharterAnchor(ctx); found {
		return anchor
	}

	return uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
}

/* CharterAge is how many blocks of decay the current puzzle has accumulated.
 *
 * Clamped rather than subtracted blind. An anchor ahead of the current height is
 * only reachable through a hand-written genesis file or a chain rolled back
 * behind an export, but in uint64 the subtraction would wrap to an enormous age
 * and read as maximally easy, which is precisely the wrong way for this to fail.
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
 * Mirrors GuildMembershipJoinProxy: derive the address from the declared pubkey,
 * require it to be the one named, verify the signature over a preimage rebuilt
 * here rather than accepted from the client, then resolve the address to a
 * player.
 *
 * The address is a subject rather than an identity, so it goes through the pure
 * GetPlayerByAddress and never GetSigningPlayer. Requiring it to resolve to the
 * founder named in the work preimage is deliberate belt-and-braces over a
 * consent that has to survive weeks of mining: addresses change hands, and a
 * key that has since been revoked and re-registered elsewhere must refuse rather
 * than found a guild for whoever holds it now.
 *
 * PermPlay on that specific address is required so a narrowly-scoped key cannot
 * sign a guild away. It is an address-level read, not a PermissionCheck, because
 * the consenting key is not the acting identity for this transaction.
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
	consentInput := types.GuildCharterConsentInput(founder.GetPlayerId(), msg.ReactorId, msg.EntrySubstationId, msg.Endpoint, anchor)

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

/* guildCharterReactorLive is the floor a reactor must clear to become a new
 * guild's primary reactor: its validator has to exist and not be jailed.
 *
 * AppendGuild writes PrimaryReactorId unconditionally, and GuildMembershipJoin
 * redelegates every joiner's infusion to whatever reactor that names, so a guild
 * founded on a tombstoned validator is a trap for its members from the moment it
 * exists. GuildUpdatePrimaryReactor holds the recovery path to exactly this bar,
 * and creation was holding it to nothing.
 *
 * Deliberately no bonded check, for the same reason that handler gives: a guild
 * may legitimately form around a validator that is still working its way into
 * the active set. The entitlement path is stricter and checks bonded itself,
 * because there the reactor's health is what is being rewarded.
 *
 * A jailed validator cannot strand a mined proof, which is what makes this safe
 * to add: the work preimage binds solver, founder and anchor and says nothing
 * about the reactor, so a solver refused here names a different reactor and
 * re-submits the same nonce.
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
 * Four conditions, and the bonded one is the load-bearing one. ReactorInitialize
 * runs from AfterValidatorCreated, so a reactor exists for every validator ever
 * created, and MsgCreateValidator picks its own min_self_delegation — a
 * validator that never joins the active set costs almost nothing, so without
 * this a hundred throwaway validators would be a hundred free guilds. Requiring
 * currently bonded caps the free supply at the size of the active set.
 *
 * This is deliberately stricter than GuildUpdatePrimaryReactor, which tolerates
 * an unbonded validator because it is a recovery path for a guild whose
 * validator is already gone. Here the whole point is rewarding a reactor that
 * works.
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
