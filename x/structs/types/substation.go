package types

import (
    math "cosmossdk.io/math"
)



func CreateEmptySubstation() (Substation) {
    return Substation{
        Power: math.ZeroInt(),
        Load: math.ZeroInt(),
    }
}

func (substation *Substation) ApplyAllocationSource(allocation Allocation) (error) {
    substation.Load = substation.Load.Add(allocation.Power.Add(allocation.TransmissionLoss))
    return nil;
}

func (substation *Substation) RemoveAllocationSource(allocation Allocation) (error) {
    substation.Load = substation.Load.Sub(allocation.Power.Add(allocation.TransmissionLoss))
    return nil;
}

func (substation *Substation) ApplyAllocationDestination(allocation Allocation) (error) {
    substation.Power = substation.Power.Add(allocation.Power)
    return nil;
}

func (substation *Substation) RemoveAllocationDestination(allocation Allocation) (error) {
    substation.Power = substation.Power.Sub(allocation.Power)
    return nil;
}