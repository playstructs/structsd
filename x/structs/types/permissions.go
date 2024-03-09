package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

type Permission uint64


const (
    // 1
	PermissionPlay Permission = 1 << iota
	// 2
	PermissionUpdate
	// 4
	PermissionDelete
	// 8
	PermissionAssets
	// 16
	PermissionAssociations
	// 32
	PermissionGrid
)

const (
    Permissionless Permission = 0 << iota
	PermissionAll = PermissionPlay | PermissionUpdate | PermissionDelete | PermissionAssets | PermissionAssociations | PermissionGrid
)

var Permission_enum = map[string]Permission {
	"permissionless":   Permissionless,
    "play":             PermissionPlay,
    "update":           PermissionUpdate,
    "delete":           PermissionDelete,
    "assets":           PermissionAssets,
	"associations":     PermissionAssociations,
    "grid":             PermissionGrid,
	"all":              PermissionAll,
}