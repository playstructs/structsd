package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)


func CreateEmptyPlayer() Player {
	return Player{
	    Id:             0,
		GuildId:        0,
		SubstationId:   0,
		Creator:        "",
	}
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

