package types

import (
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)


func CreateEmptyReactor() (Reactor) {
    return Reactor{
        Power: 0,
        Load: 0,
        PreviousStatus: Reactor_ONLINE,
        Status: Reactor_OFFLINE,
    }
}

func (reactor *Reactor) SetValidator(validator types.Validator) (error) {
	reactor.Validator =  validator.OperatorAddress
	return nil
}

func (reactor *Reactor) SetEnergy(validator types.Validator) (error) {
	reactor.Power =  validator.Tokens.Uint64()
	reactor.CheckStatus()
	return nil
}

func (reactor *Reactor) CheckStatus() {
    if (reactor.Load > reactor.Power) {
        _ = reactor.SetStatusOverload()
    } else {
        _ = reactor.SetStatusOnline()
    }
}

func (reactor *Reactor) SetStatusOnline() (error) {
    reactor.PreviousStatus = reactor.Status
    reactor.Status = Reactor_ONLINE
    return nil
}

func (reactor *Reactor) SetStatusOverload() (error) {
    reactor.PreviousStatus = reactor.Status
    reactor.Status = Reactor_OVERLOAD
    return nil
}

func (reactor *Reactor) IsOnline() bool {
    return reactor.Status == Reactor_ONLINE
}


func (reactor *Reactor) ApplyAllocationSource(allocation Allocation) (error) {
    reactor.Load = reactor.Load + (allocation.Power +  allocation.TransmissionLoss)
    reactor.CheckStatus()
    return nil;
}

func (reactor *Reactor) RemoveAllocationSource(allocation Allocation) (error) {
    reactor.Load = reactor.Load - (allocation.Power + allocation.TransmissionLoss)
    reactor.CheckStatus()
    return nil;
}