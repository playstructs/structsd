package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

func (a *Infusion) SetAddress(address string) error {

	a.Address = address

	return nil
}


func (a *Infusion) SetFuel(newFuel uint64) error {

	a.Fuel   = newFuel
	a.Energy = CalculateInfusionEnergy(a.DestinationType, newFuel)

	return nil
}


func (a *Infusion) SetDestination(destinationId uint64) error {

	a.DestinationId = destinationId

	return nil
}

func (a *Infusion) SetLinkedSourceAllocation(allocationId uint64) error {

	a.LinkedSourceAllocationId = allocationId

	return nil
}

func (a *Infusion) SetLinkedPlayerAllocation(allocationId uint64) error {

	a.LinkedPlayerAllocationId = allocationId

	return nil
}


func CalculateInfusionEnergy(destinationType ObjectType, fuel uint64) (energy uint64) {
    switch destinationType {
        case ObjectType_reactor:
            energy = CalculateReactorEnergy(fuel)
        case ObjectType_struct:
            energy = CalculateStructEnergy(fuel)
    }

    return
}

func CreateNewInfusion(destinationType ObjectType, destinationId uint64, playerAddress string, fuel uint64) Infusion {
	return Infusion{
		DestinationType: destinationType,
		DestinationId: destinationId,
		Fuel: fuel,
		Energy: CalculateInfusionEnergy(destinationType, fuel),
		Address: playerAddress,
		LinkedSourceAllocationId: 0,
		LinkedPlayerAllocationId: 0,
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

