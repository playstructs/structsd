package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
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
