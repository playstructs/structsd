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
	ErrAllocationNotFound                   = sdkerrors.Register(ModuleName, 1510, "allocation not found")
	ErrAllocationSourceType                 = sdkerrors.Register(ModuleName, 1511, "invalid source type")
	ErrAllocationSourceTypeMismatch         = sdkerrors.Register(ModuleName, 1512, "source type mismatch")
	ErrAllocationSourceNotFound             = sdkerrors.Register(ModuleName, 1513, "source not found")
	ErrAllocationSourceNotOnline            = sdkerrors.Register(ModuleName, 1514, "source not online")
	ErrAllocationConnectionChangeImpossible = sdkerrors.Register(ModuleName, 1515, "allocation connection change attempted is impossible")

    ErrPlayerRequired                       = sdkerrors.Register(ModuleName, 1530, "player account required for this action")

	ErrSubstationNotFound         = sdkerrors.Register(ModuleName, 1550, "substation not found")
	ErrSubstationHasNoPowerSource = sdkerrors.Register(ModuleName, 1551, "substation has no power source")

	ErrSubstationAvailableCapacityInsufficient = sdkerrors.Register(ModuleName, 1552, "substation capacity lower then attempted change allows for")

    ErrReactorActivation = sdkerrors.Register(ModuleName, 1571, "reactor activation failure")
	ErrReactorAvailableCapacityInsufficient = sdkerrors.Register(ModuleName, 1572, "reactor capacity lower then attempted change allows for")
)
