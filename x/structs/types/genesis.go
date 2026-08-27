package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	// this line is used by starport scaffolding # genesis/types/import
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		PortId: PortID,
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}

	// A genesis AddressList is the only way an unparseable address can reach
	// SetPlayerIndexForAddress: every transaction path derives its address from
	// a pubkey checked against PubKeyToBech32, or from an AccAddress that was
	// already parsed. Catching it here means `structsd genesis validate` fails
	// on a bad file rather than a node starting on one.
	for _, address := range gs.AddressList {
		if address == nil {
			continue
		}
		if _, err := sdk.AccAddressFromBech32(address.Address); err != nil {
			return NewAddressValidationError(address.Address, "invalid_format")
		}
	}

	// The nonce list is validated the same way and for a stronger reason: these
	// rows are what make a registration proof single-use, so a malformed or
	// duplicated entry is a replay window rather than a cosmetic problem.
	seenAddressNonce := make(map[string]struct{}, len(gs.AddressNonceList))
	for _, nonce := range gs.AddressNonceList {
		if nonce == nil {
			continue
		}
		if _, err := sdk.AccAddressFromBech32(nonce.Address); err != nil {
			return NewAddressValidationError(nonce.Address, "invalid_format")
		}
		if _, duplicate := seenAddressNonce[nonce.Address]; duplicate {
			return NewAddressValidationError(nonce.Address, "duplicate_nonce")
		}
		seenAddressNonce[nonce.Address] = struct{}{}
	}

	// GenesisImportGuild assigns the whole record onto the cache, so a genesis
	// file is the one way a guild join bypass level reaches state without
	// passing GuildCache.SetJoinInfusionMinimumBypassBy*. The readers deny an
	// undeclared level rather than trusting it, so an unvalidated file would
	// start a chain with guilds nobody can join instead of an exploitable one —
	// but failing `structsd genesis validate` on the file beats discovering it
	// as a permanently closed guild.
	for _, guild := range gs.GuildList {
		if guild.EntryRank == 0 {
			return NewParameterValidationError("guild.entryRank", guild.EntryRank, "must_be_positive")
		}
		// A name the pinned-Unicode rules reject is state no transaction could
		// produce, and InitGenesis would index it anyway; validate here so
		// `structsd genesis validate` refuses the file the migration would purge.
		if guild.Name != "" {
			if err := ValidateEntityName(guild.Name); err != nil {
				return errorsmod.Wrapf(err, "guild name (%s) on guild (%s)", guild.Name, guild.Id)
			}
		}
		if !guild.JoinInfusionMinimumBypassByRequest.IsValid() {
			return errorsmod.Wrapf(ErrInvalidGuildJoinBypassLevel, "byRequest level (%d) on guild (%s)", int32(guild.JoinInfusionMinimumBypassByRequest), guild.Id)
		}
		if !guild.JoinInfusionMinimumBypassByInvite.IsValid() {
			return errorsmod.Wrapf(ErrInvalidGuildJoinBypassLevel, "byInvite level (%d) on guild (%s)", int32(guild.JoinInfusionMinimumBypassByInvite), guild.Id)
		}
	}

	/* A provider's published duration range is a safety property, not only a
	 * policy one, and a genesis file is the only way it reaches state unchecked:
	 * GenesisImportProvider assigns the whole record onto the cache and so skips
	 * SetDurationRange, which floors the minimum at 1.
	 *
	 * Zero matters because AgreementDurationVerify is what stops a capacity
	 * resize from rescaling an agreement down to no duration at all. An agreement
	 * whose EndBlock lands on the current block is live for the rest of that
	 * block - AgreementExpirations runs in the EndBlocker, after every message -
	 * so its raised capacity is usable while the collateral still covers the old
	 * one. rescaledDuration refuses zero on its own account now, which is what
	 * covers records already on disk; this is what stops new ones.
	 */
	for _, provider := range gs.ProviderList {
		if provider.DurationMinimum == 0 {
			return NewParameterValidationError("provider.durationMinimum", provider.DurationMinimum, "must_be_positive").WithProvider(provider.Id)
		}
		if provider.DurationMinimum > provider.DurationMaximum {
			return NewParameterValidationError("provider.durationMinimum", provider.DurationMinimum, "exceeds_maximum").WithProvider(provider.Id).WithRange(0, provider.DurationMaximum)
		}
	}

	// Both of these fields are authorization input on the approve paths — the
	// join type decides which side's consent the application stands for, the
	// status decides whether it is still live — and a genesis file is the only
	// way an undeclared value reaches either. GenesisImportGuildMembershipApplication
	// assigns the whole record onto the cache, so it passes no setter that could
	// have checked, and the keeper's own writes only ever store the four declared
	// statuses.
	for _, app := range gs.GuildMembershipApplicationList {
		if !app.JoinType.IsValid() {
			return errorsmod.Wrapf(ErrGuildJoinType, "join type (%d) on application for player (%s) in guild (%s)", int32(app.JoinType), app.PlayerId, app.GuildId)
		}
		if !app.RegistrationStatus.IsValid() {
			return errorsmod.Wrapf(ErrGuildMembershipApplication, "registration status (%d) on application for player (%s) in guild (%s)", int32(app.RegistrationStatus), app.PlayerId, app.GuildId)
		}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
