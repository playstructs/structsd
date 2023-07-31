package types

import (
    sdk "github.com/cosmos/cosmos-sdk/types"
 //   "strconv"
  //  sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)



func CreateEmptySubstation() (Substation) {
    return Substation{
        PlayerConnectionAllocation: 0,
        Creator: "",
        Owner: 0,
    }
}



// TODO: Once the player construct is in place, change this section
// so that it receives a Player object. This will enforce that the
// player account exists.
func (substation *Substation) SetOwner(owner uint64) {
    substation.Owner = owner;
}



func (substation *Substation) SetCreator(creator string) {
    substation.Creator = creator;
}


// Only sets the internal variable. Does not update any of the energy draw memory values (which would need to be rebuilt).
func (substation *Substation) SetPlayerConnectionAllocation(playerConnectionAllocation uint64) {
    substation.PlayerConnectionAllocation = playerConnectionAllocation;
}



// TODO anything in this function
func (substation *Substation) IsOnline(ctx sdk.Context) (bool, error) {

    return true, nil

}