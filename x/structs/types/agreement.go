package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	//"cosmossdk.io/math"
)



func CreateBaseAgreement(creator string, owner string, providerId string, capacity uint64, startBlock uint64, endBlock uint64, allocationId string) (Agreement) {
    return Agreement{
        Creator: creator,
        Owner: owner,

        ProviderId: providerId,

        Capacity: capacity,

        StartBlock: startBlock,
        EndBlock: endBlock,

        AllocationId: allocationId,
    }
}
