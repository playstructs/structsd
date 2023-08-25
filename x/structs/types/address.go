package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//   "strconv"
	//  sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)



type AddressPermission uint64

const (
    // 1
	AddressPermissionAssociate AddressPermission = 1 << iota
    // 2
    AddressPermissionRevoke
	// 4
	AddressPermissionManageEnergy
	// 8
	AddressPermissionPlay
	// 16
	AddressPermissionManageAssets
	// 32
	AddressPermissionManagePlayer
	// 64
	AddressPermissionManageGuild

)
const (
    AddressPermissionless AddressPermission = 0 << iota
	AddressPermissionAll = AddressPermissionAssociate | AddressPermissionRevoke | AddressPermissionManageEnergy | AddressPermissionPlay|  AddressPermissionManageAssets | AddressPermissionManagePlayer | AddressPermissionManageGuild
)