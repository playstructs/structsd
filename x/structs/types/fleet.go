package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
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
            err = NewStructLocationError(structure.Type, structure.OperatingAmbit.String(), "invalid_ambit").WithStruct(structure.Id)
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
