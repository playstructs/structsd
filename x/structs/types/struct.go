package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
 *
 * This entire document is <3 garbage <3 and will
 * be re-written
 *
 */


func CheckLocation(structType StructType, locationType Object, ambit Ambit) {

}

func (structure *Struct) SetLocation(locationId string, slot uint64) error {


	structure.LocationId = locationId
	structure.Slot = slot

	return nil
}


// Take an amount of fuel and return the energy it will generate
//
// This will need some work later on to be more dynamic in
// relation to other system state, but for now it is static.
func CalculateStructPower(fuel uint64) (energy uint64, ratio uint64) {
    return fuel * StructFuelToEnergyConversion, StructFuelToEnergyConversion
}

func CreateBaseStruct(structType StructType) Struct {

    // TODO check structType


    return Struct{
        Creator:  "",
        Owner: "",

        Type: structType.Id,

        LocationId: "",
        OperatingAmbit: ,
        Slot: ,
    }
}




type StructState uint64

const (
    // 1
	StructStateBuilt State = 1 << iota
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
