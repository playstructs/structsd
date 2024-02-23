package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
    "strconv"
)


func (a *Allocation) SetPower(newPower uint64) error {

	a.Power = newPower

	return nil
}

func (a *Allocation) SetController(controller string) error {

	a.Controller = controller

	return nil
}

func (a *Allocation) Disconnect() error {

	a.DestinationId = 0

	return nil
}

func (a *Allocation) SetDestinationId(destinationSubstationId uint64) error {

	a.DestinationId = destinationSubstationId

	return nil
}

func CreateAllocation(allocationType AllocationType, sourceType ObjectType, sourceId uint64, index uint64, power uint64, creator string, controller string ) Allocation {
	return Allocation{
	    Id: GetAllocationIDString(sourceType, sourceId, index),
		Type: allocationType,
		SourceType: sourceType,
		SourceId: sourceId,
		Index: index,
		DestinationId: 0,
		Power: power,
		Creator: creator,
		Controller: controller,
		Locked: false,
	}
}


/*
 * Currently, only Players, Reactors, Structs (Power Plants), and Substations can have
 * power allocated from them to a substation.
 *
 * Use this function anytime a user is providing the objectType of the source objectType
 */
func IsValidAllocationConnectionType(objectType ObjectType) bool {
	for _, a := range []ObjectType{ObjectType_player, ObjectType_reactor, ObjectType_struct, ObjectType_substation} {
		if a == objectType {
			return true
		}
	}
	return false
}

