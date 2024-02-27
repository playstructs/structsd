package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
    //"strconv"
)

func CreateAllocationStub(allocationType AllocationType, sourceObjectId string, power uint64, creator string, controller string ) Allocation {
	return Allocation{
	    Id: "",
		Type: allocationType,
		SourceObjectId: sourceObjectId,
		Index: 0,
		DestinationObjectId: "",
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

