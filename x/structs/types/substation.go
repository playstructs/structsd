package types

import (
    sdk "github.com/cosmos/cosmos-sdk/types"
 //   "strconv"
  //  sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)



func CreateEmptySubstation() (Substation) {
    return Substation{
        Power: 0,
        Load: 0,
    }
}

func (substation *Substation) ResetPower() {
    substation.Power = 0;
}

func (substation *Substation) ApplyAllocationSource(allocation *Allocation) (error) {
    substation.Load = substation.Load + (allocation.Power + allocation.TransmissionLoss)
    return nil;
}

func (substation *Substation) RemoveAllocationSource(allocation Allocation) (error) {
    substation.Load = substation.Load - (allocation.Power + allocation.TransmissionLoss)
    return nil;
}

func (substation *Substation) ApplyAllocationDestination(allocation *Allocation) (error) {
    substation.Power = substation.Power + allocation.Power
    return nil;
}

func (substation *Substation) RemoveAllocationDestination(allocation Allocation) (error) {
    substation.Power = substation.Power - allocation.Power
    return nil;
}

func (substation *Substation) CheckStatus() (Substation_Status) {
    if (substation.Power >= substation.Load) {
        return Substation_ONLINE
    } else {
        return Substation_OFFLINE
    }

}

func (substation *Substation) IsOnline(ctx sdk.Context) (bool, error) {
    if (substation.Load == 0) {
        return true, nil
    }

/*
    var loadCheck uint64 = 0;

    if (len(substation.AllocationIn) == 0){
        substationIdString := strconv.FormatUint(substation.Id, 10)
        return false, sdkerrors.Wrapf(ErrSubstationHasNoPowerSource, "substation (%s) has no power sources", substationIdString)
    }

    if (loadCheck != 0) {
    }

*/
/*


    // Order them by size
        // At least, this would be smart to do but we're going to ignore it for now


*/

    return true, nil

}