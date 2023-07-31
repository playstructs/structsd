package types

import (
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)


func CreateEmptyReactor() (Reactor) {
    return Reactor{
        Energy: 0,
        Validator: "",
    }
}

func (reactor *Reactor) SetValidator(validator types.Validator) (error) {
	reactor.Validator =  validator.OperatorAddress
	return nil
}


// Sets the variable within the object but does not update the memory stores
func (reactor *Reactor) SetEnergy(validator types.Validator) (error) {
	reactor.Energy = validator.Tokens.Uint64()
	return nil
}
