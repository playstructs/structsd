package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//   "strconv"
	//  sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func CreateEmptyAddress() Address {
	return Address{

	}
}

func (address *Address) SetId(id uint64) error {

	address.Id = id

	return nil
}



type AddressPermission uint16

const (
    // 1
	AddressPermissionGrantAssociate AddressPermission = 1 << iota
    // 2
	AddressPermissionAssociate
    // 4
    AddressPermissionGrantRevoke
    // 8
    AddressPermissionRevoke
    // 16
	AddressPermissionGrantManageEnergy
	// 32
	AddressPermissionManageEnergy
	// 64
	AddressPermissionGrantPlay
	// 128
	AddressPermissionPlay
	// 256
	AddressPermissionGrantManageAssets
	// 512
	AddressPermissionManageAssets

)
const (
    AddressPermissionless AddressPermission = 0 << iota
	AddressPermissionAll = AddressPermissionAssociate | AddressPermissionRevoke | AddressPermissionManageEnergy | AddressPermissionGrantPlay | AddressPermissionManageAssets
	AddressPermissionAllWithGrant = AddressPermissionGrantAssociate | AddressPermissionAssociate | AddressPermissionGrantRevoke | AddressPermissionRevoke | AddressPermissionGrantManageEnergy | AddressPermissionManageEnergy | AddressPermissionGrantPlay | AddressPermissionPlay | AddressPermissionGrantManageAssets | AddressPermissionManageAssets
)