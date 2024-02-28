package types

import (
	"cosmossdk.io/math"
)

func CreateEmptyReactor() Reactor {
	return Reactor{
		Load:       0,
		Capacity:   0,
		Fuel:       0,
		Validator: "",
		RawAddress: nil,
		DefaultCommission: math.LegacyZeroDec(),
        GuildId: "",
	}
}






// Take an amount of fuel and return the energy it will generate
//
// This will need some work later on to be more dynamic in
// relation to other system state, but for now it is static.
func CalculateReactorPower(fuel uint64) (energy uint64) {
    return fuel * ReactorFuelToEnergyConversion
}

type ReactorPermission Permission

const (
    // 1
	ReactorPermissionAllocate ReactorPermission = 1 << iota
	// 2
	ReactorPermissionUpdateGuild
	// 4
	ReactorPermissionUpdateAllocationRules
)
const (
    ReactorPermissionless ReactorPermission = 0 << iota
	ReactorPermissionAll = ReactorPermissionAllocate | ReactorPermissionUpdateGuild | ReactorPermissionUpdateAllocationRules
)