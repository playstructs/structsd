package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

func (planet *Planet) SetCreator(creator string) error {
	planet.Creator = creator
	return nil
}



func (planet *Planet) SetOwner(playerId string) error {
	planet.Owner = playerId
	return nil
}


func (planet *Planet) SetStatus(status uint64) error {
	planet.Status = status
	return nil
}


func (planet *Planet) SetLandSlot(structure Struct) error {
    planet.Land[structure.Slot] = structure.Id
	return nil
}



func CreateEmptyPlanet() Planet {

    defaultEmptySlots := []string{"", "", "", ""}

	return Planet{
		Creator:  "",
		Owner: "",
		Status: 0,

        Space: defaultEmptySlots,
        Sky: defaultEmptySlots,
        Land: defaultEmptySlots,
        Water: defaultEmptySlots,

		// TODO make these values dynamic some day
		MaxOre:     PlanetStartingOre,
		SpaceSlots: PlanetStartingSlots,
        SkySlots:   PlanetStartingSlots,
        LandSlots:  PlanetStartingSlots,
        WaterSlots: PlanetStartingSlots,


	}
}
