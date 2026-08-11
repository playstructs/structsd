package types

import (
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

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
