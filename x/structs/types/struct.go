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




func (structure *Struct) SetStatus(status string) error {
	structure.Status = status
	return nil
}


func (structure *Struct) SetMiningSystemStatus(status string) error {
	structure.MiningSystemStatus = status
	return nil
}


func (structure *Struct) SetRefiningSystemStatus(status string) error {
	structure.RefiningSystemStatus = status
	return nil
}


func (structure *Struct) SetSlot(slot uint64) error {
	structure.Slot = slot
	return nil
}

func (structure *Struct) SetPlanetId(planetId string) error {
	structure.PlanetId = planetId
	return nil
}


func (structure *Struct) SetBuildStartBlock(block uint64) error {

    structure.BuildStartBlock = block

    return nil
}

func (structure *Struct) SetMiningSystemActivationBlock(block uint64) error {
    structure.ActiveMiningSystemBlock = block

    return nil
}

func (structure *Struct) SetRefiningSystemActivationBlock(block uint64) error {
    structure.ActiveRefiningSystemBlock = block

    return nil
}

// Take an amount of fuel and return the energy it will generate
//
// This will need some work later on to be more dynamic in
// relation to other system state, but for now it is static.
func CalculateStructPower(fuel uint64) (energy uint64, ratio uint64) {
    return fuel * StructFuelToEnergyConversion, StructFuelToEnergyConversion
}

func CreateBaseStruct(structType string) Struct {

    var newCategory         string
    var newType             string
    var newAmbit            string

    var newMiningSystem     uint64
    var newRefiningSystem   uint64
    var newPowerSystem      uint64

    var newPassiveDraw              uint64
    var newActiveMiningSystemDraw   uint64
    var newActiveRefiningSystemDraw uint64

    switch structType {
    	case "Mining Rig":
	    	newCategory = "Planetary Struct"
	    	newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 1;
            newRefiningSystem = 0;
            newPowerSystem    = 0;

            newPassiveDraw              = 10;
            newActiveMiningSystemDraw   = 20;
            newActiveRefiningSystemDraw = 0;

	    case "Refinery":
	        newCategory = "Planetary Struct"
		    newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 0;
            newRefiningSystem = 1;
            newPowerSystem    = 0;

            newPassiveDraw              = 20;
            newActiveMiningSystemDraw   = 0;
            newActiveRefiningSystemDraw = 30;

	    case "Small Generator":
	        newCategory = "Planetary Struct"
		    newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 0;
            newRefiningSystem = 0;
            newPowerSystem    = 1;

            newPassiveDraw              = 5;
            newActiveMiningSystemDraw   = 0;
            newActiveRefiningSystemDraw = 0;

	    default:
    }

	return Struct{
		Creator:  "",
		Owner: "",
		Status: "BUILDING",

		MaxHealth: 3,
		Health: 3,

		Type: newType,
        Category: newCategory,
        Ambit: newAmbit,
        Slot: 0,

        MiningSystem: newMiningSystem,
        RefiningSystem: newRefiningSystem,
        PowerSystem: newPowerSystem,

        PassiveDraw: newPassiveDraw,
        ActiveMiningSystemDraw: newActiveMiningSystemDraw,
        ActiveRefiningSystemDraw: newActiveRefiningSystemDraw,

        MiningSystemStatus: "INACTIVE",
        ActiveMiningSystemBlock: 0,

        RefiningSystemStatus: "INACTIVE",
        ActiveRefiningSystemBlock: 0,

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
