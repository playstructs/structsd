package types

import (
    sdk "github.com/cosmos/cosmos-sdk/types"

)


func (a *Allocation) SetPower(ctx sdk.Context, proposal AllocationProposal) (error) {

    a.Power = proposal.Power

    //TODO: Change into a parameter
    a.TransmissionLoss = proposal.Power / 4

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

