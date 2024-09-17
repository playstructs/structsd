package types

import (
	"cosmossdk.io/math"
)

func CreateEmptyReactor() Reactor {
	return Reactor{
		Validator: "",
		RawAddress: nil,
		DefaultCommission: math.LegacyZeroDec(),
        GuildId: "",
	}
}
