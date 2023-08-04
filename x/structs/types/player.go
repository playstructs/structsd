package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)


func (player *Player) SetId(id uint64) error {

	player.Id = id

	return nil
}


func (player *Player) SetGuild(guildId uint64) error {

	player.GuildId = guildId

	return nil
}


func (player *Player) SetSubstation(substationId uint64) error {

	player.SubstationId = substationId

	return nil
}


func CreateEmptyPlayer() Player {
	return Player{
	    Id:             0,
		GuildId:        0,
		SubstationId:   0,
	}
}



type PlayerPermission uint16

const (
    // 1
	PlayerPermissionGrantUpdate PlayerPermission = 1 << iota
    // 2
	PlayerPermissionUpdate
	// 4
	PlayerPermissionGrantRegisterPlayer
	// 8
	PlayerPermissionRegisterPlayer
	// 16
	PlayerPermissionGrantDelete
	// 32
	PlayerPermissionDelete
)
const (
    PlayerPermissionless PlayerPermission = 0 << iota
	PlayerPermissionAll = PlayerPermissionUpdate | PlayerPermissionRegisterPlayer | PlayerPermissionDelete
	PlayerPermissionAllWithGrant = PlayerPermissionGrantUpdate | PlayerPermissionUpdate | PlayerPermissionGrantRegisterPlayer | PlayerPermissionRegisterPlayer | PlayerPermissionGrantDelete | PlayerPermissionDelete
)

