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

