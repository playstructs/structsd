package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
    "strconv"
)

func (a *Allocation) SetCreator(creator string) error {

	a.Creator = creator

	return nil
}


func (a *Allocation) SetPower(newPower uint64) error {

	a.Power = newPower

	return nil
}

func (a *Allocation) SetController(controller string) error {

	a.Controller = controller

	return nil
}

func (a *Allocation) SetSource(sourceId uint64) error {

	a.SourceId = sourceId

	return nil
}



func (a *Allocation) Disconnect() error {

	a.DestinationId = 0

	return nil
}

func (a *Allocation) Connect(destinationSubstationId uint64) error {

	a.DestinationId = destinationSubstationId

	return nil
}

func (a *Allocation) SetLinkedInfusion(address string) error {
    a.HasLinkedInfusion = true

    sourceId := strconv.FormatUint(a.SourceId , 10)
    a.LinkedInfusion = a.SourceType.String() + "-" + sourceId + "-" + address

    return nil
}

func (a *Allocation) ClearLinkedInfusion() error {
    a.HasLinkedInfusion = false
    a.LinkedInfusion = ""

    return nil
}


func CreateEmptyAllocation(sourceType ObjectType) Allocation {
	return Allocation{
		Id: 0,
		SourceType: sourceType,
		SourceId: 0,
		DestinationId: 0,
		Power: 0,
		Creator: "",
		Controller: "",
		Locked: false,
		HasLinkedInfusion: false,
		LinkedInfusion: "",
	}
}



/*
 * Currently, only Reactors and Structs (Power Plants) can have
 * power allocated from them to a substation.
 *
 * Substations cannot connect to Substations. ObjectType_substation would need
 * be added to the list below to enable such a connection.
 *
 * Use this function anytime a user is providing the objectType of the source objectType
 */
func IsValidAllocationConnectionType(objectType ObjectType) bool {
	for _, a := range []ObjectType{ObjectType_reactor, ObjectType_struct, ObjectType_substation} {
		if a == objectType {
			return true
		}
	}
	return false
}

