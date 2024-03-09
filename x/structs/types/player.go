package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)


func (player *Player) IsOnline() (online bool){
    if ((player.Load + player.StructsLoad) <= (player.Capacity + player.CapacitySecondary)) {
        online = true
    } else {
        online = false
    }
    return
}

func (player *Player) CanSupportNewLoad(newLoad uint64) (online bool){
    if ((player.Load + player.StructsLoad + newLoad) <= (player.Capacity + player.CapacitySecondary)) {
        online = true
    } else {
        online = false
    }
    return
}

