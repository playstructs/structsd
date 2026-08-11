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

	// GenesisImportGuild assigns the whole record onto the cache, so a genesis
	// file is the one way a guild join bypass level reaches state without
	// passing GuildCache.SetJoinInfusionMinimumBypassBy*. The readers deny an
	// undeclared level rather than trusting it, so an unvalidated file would
	// start a chain with guilds nobody can join instead of an exploitable one —
	// but failing `structsd genesis validate` on the file beats discovering it
	// as a permanently closed guild.
	for _, guild := range gs.GuildList {
		if !guild.JoinInfusionMinimumBypassByRequest.IsValid() {
			return errorsmod.Wrapf(ErrInvalidGuildJoinBypassLevel, "byRequest level (%d) on guild (%s)", int32(guild.JoinInfusionMinimumBypassByRequest), guild.Id)
		}
		if !guild.JoinInfusionMinimumBypassByInvite.IsValid() {
			return errorsmod.Wrapf(ErrInvalidGuildJoinBypassLevel, "byInvite level (%d) on guild (%s)", int32(guild.JoinInfusionMinimumBypassByInvite), guild.Id)
		}
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
