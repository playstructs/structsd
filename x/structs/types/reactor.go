package types

import (
	"cosmossdk.io/math"
)

func CreateEmptyReactor() Reactor {
	return Reactor{
		Energy:    0,
		Fuel:       0,
		Validator: "",
		RawAddress: nil,
		Activated: false,
		AutomatedAllocations: true,
		AllowManualAllocations: false,
		AllowExternalAllocations: false,
		AllowUncappedAllocations: false,
		DelegateMinimumBeforeAllowedAllocations: math.LegacyOneDec(),
		DelegateTaxOnAllocations: math.LegacyZeroDec(),
		ServiceSubstationId: 0,

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

func (reactor *Reactor) SetRawAddress(rawAddress []byte) {
	reactor.RawAddress = rawAddress
}

func (reactor *Reactor) SetId(id uint64) {
	reactor.Id = id
}

func (reactor *Reactor) SetServiceSubstationId(serviceSubstationId uint64) {
	reactor.ServiceSubstationId = serviceSubstationId
}

// Take an amount of fuel and return the energy it will generate
//
// This will need some work later on to be more dynamic in
// relation to other system state, but for now it is static.
func CalculateReactorEnergy(fuel uint64) (energy uint64) {
    return fuel * ReactorFuelToEnergyConversion
}

type ReactorPermission uint64

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