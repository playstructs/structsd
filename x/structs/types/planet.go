package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

func (planet *Planet) SetCreator(creator string) error {

	planet.Creator = creator

	return nil
}



func (planet *Planet) SetOwner(playerId uint64) error {

	planet.Owner = playerId

	return nil
}


func (planet *Planet) SetStatus(status uint64) error {

	planet.Status = status

	return nil
}

func CreateEmptyPlanet() Planet {

    defaultEmptySlots := []uint64{0, 0, 0, 0}

	return Planet{
		Creator:  "",
		Owner: 0,
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
