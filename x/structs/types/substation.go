package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	//   "strconv"
	//  sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func CreateEmptySubstation() Substation {
	return Substation{
		PlayerConnectionAllocation: 0,
		Creator:                    "",
		Owner:                      0,
	}
}

func (substation *Substation) SetId(id uint64) error {

	substation.Id = id

	return nil
}

func (substation *Substation) HasPlayerCapacity() bool {
    return ((substation.Energy + substation.PlayerConnectionAllocation) <= substation.Load)
}

// TODO: Once the player construct is in place, change this section
// so that it receives a Player object. This will enforce that the
// player account exists.
func (substation *Substation) SetOwner(owner uint64) {
	substation.Owner = owner
}

func (substation *Substation) SetCreator(creator string) {
	substation.Creator = creator
}

// Only sets the internal variable. Does not update any of the energy draw memory values (which would need to be rebuilt).
func (substation *Substation) SetPlayerConnectionAllocation(playerConnectionAllocation uint64) {
	substation.PlayerConnectionAllocation = playerConnectionAllocation
}

// TODO anything in this function
func (substation *Substation) IsOnline(ctx sdk.Context) (bool, error) {

	return true, nil

}



type SubstationPermission uint64

const (
    // 1
	SubstationPermissionConnectPlayer SubstationPermission = 1 << iota
    // 2
    SubstationPermissionDisconnectPlayer
    // 4
	SubstationPermissionConnectAllocation
	// 8
	SubstationPermissionDisconnectAllocation
	// 16
	SubstationPermissionAllocate
	// 32
	SubstationPermissionDelete
)
const (
    SubstationPermissionless SubstationPermission = 0 << iota
	SubstationPermissionAll = SubstationPermissionConnectPlayer | SubstationPermissionDisconnectPlayer | SubstationPermissionConnectAllocation | SubstationPermissionDisconnectAllocation | SubstationPermissionAllocate | SubstationPermissionDelete
)