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
