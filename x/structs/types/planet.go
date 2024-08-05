package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
)

func (planet *Planet) SetCreator(creator string) error {
	planet.Creator = creator
	return nil
}



func (planet *Planet) SetOwner(playerId string) error {
	planet.Owner = playerId
	return nil
}


func (planet *Planet) SetStatus(status PlanetStatus) error {
	planet.Status = status
	return nil
}


func (planet *Planet) SetSlot(structure Struct) (err error) {

    switch structure.OperatingAmbit {
        case Ambit_water:
            planet.Water[structure.Slot] = structure.Id
        case Ambit_land:
            planet.Land[structure.Slot]  = structure.Id
        case Ambit_air:
            planet.Air[structure.Slot]   = structure.Id
        case Ambit_space:
            planet.Space[structure.Slot] = structure.Id
        default:
            err = sdkerrors.Wrapf(ErrStructAction, "Struct cannot exist in the defined ambit (%s) ", structure.OperatingAmbit)
    }

	return
}



func CreateEmptyPlanet() Planet {

    defaultEmptySlots := []string{"", "", "", ""}

	return Planet{
		Creator:  "",
		Owner: "",
		Status: PlanetStatus_active,

        Space: defaultEmptySlots,
        Air:   defaultEmptySlots,
        Land:  defaultEmptySlots,
        Water: defaultEmptySlots,

		// TODO make these values dynamic some day
		MaxOre:     PlanetStartingOre,
		SpaceSlots: PlanetStartingSlots,
        AirSlots:   PlanetStartingSlots,
        LandSlots:  PlanetStartingSlots,
        WaterSlots: PlanetStartingSlots,

	}
}
