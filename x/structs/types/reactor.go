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

func (reactor *Reactor) SetValidator(validator types.Validator) (error) {
	reactor.Validator =  validator.OperatorAddress
	return nil
}

func (reactor *Reactor) SetEnergy(validator types.Validator) (error) {
	reactor.Power =  validator.Tokens
	return nil
}

func (reactor *Reactor) SetStatusOnline() (error) {
    reactor.Status = Reactor_ONLINE
    return nil
}

func (reactor *Reactor) SetStatusOverload() (error) {
    reactor.Status = Reactor_OVERLOAD
    return nil
}

func (reactor *Reactor) ApplyAllocationSource(allocation Allocation) (error) {
    reactor.Load = reactor.Load.Add(allocation.Power.Add(allocation.TransmissionLoss))
    return nil;
}

func (reactor *Reactor) RemoveAllocationSource(allocation Allocation) (error) {
    reactor.Load = reactor.Load.Sub(allocation.Power.Add(allocation.TransmissionLoss))
    return nil;
}