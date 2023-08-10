package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

func (a *Infusion) SetAddress(address string) error {

	a.Address = address

	return nil
}


func (a *Infusion) SetFuel(newFuel uint64) error {

	a.Fuel = newFuel

	return nil
}


func (a *Infusion) SetDestination(destinationId uint64) error {

	a.DestinationId = destinationId

	return nil
}

func (a *Infusion) SetLinkedAllocation(allocationId uint64) error {

	a.LinkedAllocation = allocationId

	return nil
}



func CreateNewInfusion(sourceType ObjectType, destinationId uint64, playerAddress string, fuel uint64) Infusion {
	return Infusion{
		DestinationType: sourceType,
		DestinationId: destinationId,
		Fuel: fuel,
		Address: playerAddress,
		LinkedAllocation: 0,
	}
}



/*
 * Only Reactors and Structs (Power Plants) can have Alpha infused
 *
 * Use this function anytime a user is providing the objectType of the source objectType
 */
func IsValidInfusionConnectionType(objectType ObjectType) bool {
	for _, a := range []ObjectType{ObjectType_reactor, ObjectType_struct} {
		if a == objectType {
			return true
		}
	}
	return false
}

