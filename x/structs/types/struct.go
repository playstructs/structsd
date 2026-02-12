package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
)

/*
 *
 * This entire document is <3 garbage <3 and will
 * be re-written
 *
 */




func CreateBaseStruct(structTypeId uint64, creator string, owner string, locationType ObjectType, ambit Ambit) (Struct) {
    return Struct{
        Creator: creator,
        Owner: owner,

        Type: structTypeId,

        LocationType: locationType,
        OperatingAmbit: ambit,
    }
}




type StructState uint64

const (
    // 1
	StructStateMaterialized StructState = 1 << iota
	// 2
	StructStateBuilt
	// 4
	StructStateOnline
	// 8
	StructStateStored
	// 16
	StructStateHidden
	// 32
    StructStateDestroyed
    // 64
    StructStateLocked // Unsure if needed
)

const (
    StructStateless StructState = 0 << iota
	StructStateAll = StructStateMaterialized | StructStateBuilt | StructStateOnline | StructStateStored | StructStateHidden | StructStateLocked
)


var StructState_enum = map[string]StructState {
	"stateless":    StructStateless,
	"materialized": StructStateMaterialized,
    "built":        StructStateBuilt,
    "online":       StructStateOnline,
    "stored":       StructStateStored,
    "hidden":       StructStateHidden,
    "destroyed":    StructStateDestroyed,
    "locked":       StructStateLocked,
	"all":          StructStateAll,
}

