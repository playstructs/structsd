package types

import (
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	"cosmossdk.io/math"
)

func CreateEmptyReactor() Reactor {
	return Reactor{
		Energy:    0,
		Validator: "",
		Activated: false,

		AutomatedAllocations: true,
		AllowManualAllocations: false,
		AllowExternalAllocations: false,
		AllowUncappedAllocations: false,
		DelegateMinimumBeforeAllowedAllocations: math.LegacyOneDec(),
		DelegateTaxOnAllocations: math.LegacyZeroDec(),

	}
}


func (reactor *Reactor) SetActivated(activated bool) error {
	reactor.Activated = activated
	return nil
}

func (reactor *Reactor) SetValidator(validatorAddress string) error {
	reactor.Validator = validatorAddress
	return nil
}

func (reactor *Reactor) SetId(id uint64) {
	reactor.Id = id
}

// Sets the variable within the object but does not update the memory stores
func (reactor *Reactor) SetEnergy(validator staking.Validator) error {
	reactor.Energy = validator.Tokens.Uint64()
	return nil
}


type ReactorPermission uint16

const (
    // 1
	ReactorPermissionGrantAllocate ReactorPermission = 1 << iota
    // 2
	ReactorPermissionAllocate
	// 4
	ReactorPermissionGrantUpdateGuild
	// 8
	ReactorPermissionUpdateGuild
	// 16
	ReactorPermissionGrantUpdateAllocationRules
	// 32
	ReactorPermissionUpdateAllocationRules
)
const (
    ReactorPermissionless ReactorPermission = 0 << iota
	ReactorPermissionAll = ReactorPermissionAllocate | ReactorPermissionUpdateGuild | ReactorPermissionUpdateAllocationRules
	ReactorPermissionAllWithGrant = ReactorPermissionGrantAllocate | ReactorPermissionAllocate | ReactorPermissionGrantUpdateGuild | ReactorPermissionUpdateGuild | ReactorPermissionGrantUpdateAllocationRules | ReactorPermissionUpdateAllocationRules
)