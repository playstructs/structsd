package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/structs module sentinel errors
var (
	ErrSample               = sdkerrors.Register(ModuleName, 1100, "sample error")
	ErrInvalidPacketTimeout = sdkerrors.Register(ModuleName, 1500, "invalid packet timeout")
	ErrInvalidVersion       = sdkerrors.Register(ModuleName, 1501, "invalid version")
)


var (
    ErrAllocationNotFound         = sdkerrors.Register(ModuleName, 1510, "allocation not found")
    ErrAllocationSourceType       = sdkerrors.Register(ModuleName, 1511, "invalid source type")
    ErrAllocationSourceNotFound   = sdkerrors.Register(ModuleName, 1512, "source not found")

    ErrSubstationNotFound         = sdkerrors.Register(ModuleName, 1550, "substation not found")
)
