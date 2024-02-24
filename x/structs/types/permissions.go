package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

type Permission uint64

const (
    Permissionless Permission = 0 << iota
)

