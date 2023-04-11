package types

import (
    math "cosmossdk.io/math"
    sdk "github.com/cosmos/cosmos-sdk/types"

)


func (a *Allocation) SetPower(ctx sdk.Context, proposal AllocationProposal) (error) {

    a.Power = proposal.Power

    //TODO: Change into a parameter
    transmissionLossBase, _ := math.NewIntFromString("4");
    a.TransmissionLoss = proposal.Power.Quo(transmissionLossBase)

    return nil
}

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

