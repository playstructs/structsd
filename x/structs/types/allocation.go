package types

import (
    sdk "github.com/cosmos/cosmos-sdk/types"

)


func (a *Allocation) SetPower(ctx sdk.Context, newPower uint64) (error) {

    a.Power = newPower

    //TODO: Change into a parameter
    a.TransmissionLoss = newPower / 4

    return nil
}

func (a *Allocation) Disconnect(ctx sdk.Context) (error) {

    a.DestinationId = nil

    return nil
}

func (a *Allocation) Connect(ctx sdk.Context, destinationId uint64) (error) {

    a.DestinationId = destinationId

    return nil
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
func IsValidAllocationConnectionType(objectType ObjectType) (bool) {
	for _, a := range []ObjectType{ObjectType_reactor, ObjectType_struct} {
		if a == objectType {
			return true
		}
	}
	return false
}

