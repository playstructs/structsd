package types

import (

)

/*
 * Currently, only Reactors, Structs (Power Plants), and Substations can have
 * power allocated from them to a substation.
 *
 * Use this function anytime a user is providing the objectType of the source objectType
 */
func IsValidAllocationConnectionType(objectType ObjectType) (bool) {
	for _, a := range []ObjectType{ObjectType_reactor, ObjectType_substation, ObjectType_struct} {
		if a == objectType {
			return true
		}
	}
	return false
}

