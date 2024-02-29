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

type PlayerPermission Permission

const (
    // 1
	PlayerPermissionUpdate PlayerPermission = 1 << iota
	// 2
	PlayerPermissionDelete
	// 4
	PlayerPermissionSubstation

)
const (
    PlayerPermissionless PlayerPermission = 0 << iota
	PlayerPermissionAll = PlayerPermissionUpdate |  PlayerPermissionDelete | PlayerPermissionSubstation
)

