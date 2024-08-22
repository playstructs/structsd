package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
)

/*
 *
 * This entire document is <3 garbage <3 and will
 * be re-written
 *
 */


func CheckBuildLocation(structType StructType, locationType ObjectType, ambit Ambit) (habitable bool, err error ) {

    // A little overly complicated at the moment but can
    // easily be expanded to allow for Structs to be built
    // on other objects
    //
    // Currently you can only build fleet and planet structs on planets
    switch locationType {
        case ObjectType_planet:
            if (structType.Category == ObjectType_planet || structType.Category == ObjectType_fleet) {
                habitable = true
            }
        default:
            habitable = false
            err = sdkerrors.Wrapf(ErrStructAction, "Struct cannot be exist in the defined location (%s) ", locationType)
    }

    // Check that the Struct can exist in the specified ambit
    if Ambit_flag[ambit]&structType.PossibleAmbit != 0 {
        habitable = true
    } else {
        habitable = false
        err = sdkerrors.Wrapf(ErrStructAction, "Struct cannot be exist in the defined ambit (%s) based on structType (%d) ", ambit, structType.Id)
    }

    return

}

func (structure *Struct) SetLocation(locationId string, slot uint64) error {


	structure.LocationId = locationId
	structure.Slot = slot

	return nil
}


func CreateBaseStruct(structType StructType, creator string, owner string, locationId string, locationType ObjectType, ambit Ambit, slot uint64) (Struct, error) {

    _, err := CheckBuildLocation(structType, locationType, ambit)

    return Struct{
        Creator: creator,
        Owner: owner,

        Type: structType.Id,

        LocationType: locationType,
        LocationId: locationId,
        OperatingAmbit: ambit,
        Slot: slot,
    }, err
}




type StructState uint64

const (
    // 1
	StructStateBuilt StructState = 1 << iota
	// 2
	StructStateOnline
	// 4
	StructStateStored
	// 8
	StructStateStealth
	// 16
    StructStateDestroyed
    // 32
    StructStateLocked // Unsure if needed
)

const (
    StructStateless StructState = 0 << iota
	StructStateAll = StructStateBuilt | StructStateOnline | StructStateStored | StructStateStealth | StructStateLocked
)


var StructState_enum = map[string]StructState {
	"stateless":    StructStateless,
    "built":        StructStateBuilt,
    "online":       StructStateOnline,
    "stored":       StructStateStored,
    "stealth":      StructStateStealth,
    "destroyed":    StructStateDestroyed,
    "locked":       StructStateLocked,
	"all":          StructStateAll,
}

