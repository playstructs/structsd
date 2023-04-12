package types

import (
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)


func (r *Reactor) SetEnergy(validator types.Validator) (error) {
	r.Power =  validator.Tokens
	return nil
}

func (r *Reactor) SetStatusOnline() (error) {
    r.Status = Reactor_ONLINE
    return nil
}

func (r *Reactor) SetStatusOverload() (error) {
    r.Status = Reactor_OVERLOAD
    return nil
}

func (r *Reactor) ApplyAllocationSource(allocation Allocation) (error) {
    r.Load = r.Load.Add(allocation.Power.Add(allocation.TransmissionLoss))
    return nil;
}