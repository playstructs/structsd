package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)


func (player *Player) SetId(id uint64) error {

	player.Id = id

	return nil
}

func (player *Player) SetCreator(creator string) error {

	player.Creator = creator

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
		Creator:        "",
	}
}



type PlayerPermission uint64

const (
    // 1
	PlayerPermissionGrantUpdate PlayerPermission = 1 << iota
    // 2
	PlayerPermissionUpdate
	// 4
	PlayerPermissionGrantDelete
	// 8
	PlayerPermissionDelete
)
const (
    PlayerPermissionless PlayerPermission = 0 << iota
	PlayerPermissionAll = PlayerPermissionUpdate |  PlayerPermissionDelete
	PlayerPermissionAllWithGrant = PlayerPermissionGrantUpdate | PlayerPermissionUpdate | PlayerPermissionGrantDelete | PlayerPermissionDelete
)

