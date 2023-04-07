package types

import (

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)


func (r Reactor) SetEnergy(ctx sdk.Context, validator types.Validator) (error) {
	r.Power = validator.Tokens
	return nil
}

func (r Reactor) SetStatusOnline(ctx sdk.Context) (error) {
    r.Status = Reactor_ONLINE
    return nil
}

func (r Reactor) SetStatusOverload(ctx sdk.Context) (error) {
    r.Status = Reactor_OVERLOAD
    return nil
}