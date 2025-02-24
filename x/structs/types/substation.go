package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//   "strconv"
	//  sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)


func CreateBaseSubstation(creator string, owner string) (Substation) {
    return Substation{
        Creator: creator,
        Owner: owner,
    }
}
