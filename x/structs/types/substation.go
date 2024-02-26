package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	//   "strconv"
	//  sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type SubstationPermission Permission

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
	// 64
	SubstationPermissionRouteGuild
)
const (
    SubstationPermissionless SubstationPermission = 0 << iota
	SubstationPermissionAll = SubstationPermissionConnectPlayer | SubstationPermissionDisconnectPlayer | SubstationPermissionConnectAllocation | SubstationPermissionDisconnectAllocation | SubstationPermissionAllocate | SubstationPermissionDelete | SubstationPermissionRouteGuild
)