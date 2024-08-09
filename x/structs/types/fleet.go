package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
)

func (fleet *Fleet) SetSlot(structure Struct) (err error) {

    switch structure.OperatingAmbit {
        case Ambit_water:
            fleet.Water[structure.Slot] = structure.Id
        case Ambit_land:
            fleet.Land[structure.Slot]  = structure.Id
        case Ambit_air:
            fleet.Air[structure.Slot]   = structure.Id
        case Ambit_space:
            fleet.Space[structure.Slot] = structure.Id
        default:
            err = sdkerrors.Wrapf(ErrStructAction, "Struct cannot exist in the defined ambit (%s) ", structure.OperatingAmbit)
    }

	return
}



func CreateEmptyFleet() Fleet {

    defaultEmptySlots := []string{"", "", "", ""}

	return Fleet{
		Owner: "",

        Space: defaultEmptySlots,
        Air:   defaultEmptySlots,
        Land:  defaultEmptySlots,
        Water: defaultEmptySlots,

		SpaceSlots: PlanetStartingSlots,
        AirSlots:   PlanetStartingSlots,
        LandSlots:  PlanetStartingSlots,
        WaterSlots: PlanetStartingSlots,

	}
}
