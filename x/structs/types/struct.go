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

func (structure *Struct) SetCreator(creator string) error {

	structure.Creator = creator

	return nil
}



func (structure *Struct) SetOwner(playerId uint64) error {

	structure.Owner = playerId

	return nil
}


func (structure *Struct) SetStatus(status string) error {

	structure.Status = status

	return nil
}

func (structure *Struct) SetSlot(slot uint64) error {

	structure.Slot = slot

	return nil
}

func (structure *Struct) SetBuildStartBlock(block uint64) error {

    structure.BuildStartBlock = block

    return nil
}


func CreateBaseStruct(structType string) Struct {

    var newCategory string
    var newType string
    var newAmbit string

    var newMiningSystem uint64
    var newRefiningSystem uint64
    var newPowerSystem uint64

    var newPassiveDraw uint64

    switch structType {
    	case "Mining Rig":
	    	newCategory = "Planetary Struct"
	    	newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 1;
            newRefiningSystem = 0;
            newPowerSystem    = 0;

            newPassiveDraw    = 10;

	    case "Refinery":
	        newCategory = "Planetary Struct"
		    newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 0;
            newRefiningSystem = 1;
            newPowerSystem    = 0;

            newPassiveDraw    = 20;

	    case "Small Generator":
	        newCategory = "Planetary Struct"
		    newType     = structType
            newAmbit    = "LAND"

            newMiningSystem   = 0;
            newRefiningSystem = 0;
            newPowerSystem    = 1;

            newPassiveDraw    = 5;

	    default:
    }

	return Struct{
		Creator:  "",
		Owner: 0,
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

	}
}
