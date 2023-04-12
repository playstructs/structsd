package types

import (
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	math "cosmossdk.io/math"
)


func CreateEmptyReactor() (Reactor) {
    return Reactor{
        Power: math.ZeroInt(),
        Load: math.ZeroInt(),
        Status: Reactor_OFFLINE,
    }
}

func (r *Reactor) SetValidator(validator types.Validator) (error) {
	r.Validator =  validator.OperatorAddress
	return nil
}

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