package types

import (
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func CreateEmptyReactor() Reactor {
	return Reactor{
		Energy:    0,
		Validator: "",
		Activated: false,
	}
}

func (reactor *Reactor) SetActivated(activated bool) error {
	reactor.Activated = activated
	return nil
}

func (reactor *Reactor) SetValidator(validatorAddress string) error {
	reactor.Validator = validatorAddress
	return nil
}

func (reactor *Reactor) SetId(id uint64) {
	reactor.Id = id
}

// Sets the variable within the object but does not update the memory stores
func (reactor *Reactor) SetEnergy(validator staking.Validator) error {
	reactor.Energy = validator.Tokens.Uint64()
	return nil
}


type ReactorPermission uint16

const (
    // 1
	ReactorPermissionGrantAllocate ReactorPermission = 1 << iota
    // 2
	ReactorPermissionAllocate
	// 4
	ReactorPermissionGrantUpdateGuild
	// 8
	ReactorPermissionUpdateGuild
)
const (
    ReactorPermissionless ReactorPermission = 0 << iota
	ReactorPermissionAll = ReactorPermissionAllocate | ReactorPermissionUpdateGuild
	ReactorPermissionAllWithGrant = ReactorPermissionGrantAllocate | ReactorPermissionAllocate | ReactorPermissionGrantUpdateGuild | ReactorPermissionUpdateGuild
)