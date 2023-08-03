package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

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
	return Guild{
		GuildId:        0,
		SubstationId:   0,
	}
}



type GuildPermission uint16

const (
    // 1
	GuildPermissionGrantUpdate GuildPermission = 1 << iota
    // 2
	GuildPermissionUpdate
	// 4
	GuildPermissionGrantRegisterPlayer
	// 8
	GuildPermissionRegisterPlayer
	// 16
	GuildPermissionGrantDelete
	// 32
	GuildPermissionDelete
)
const (
    GuildPermissionless GuildPermission = 0 << iota
	GuildPermissionAll = GuildPermissionUpdate | GuildPermissionRegisterPlayer | GuildPermissionDelete
	GuildPermissionAllWithGrant = GuildPermissionGrantUpdate | GuildPermissionUpdate | GuildPermissionGrantRegisterPlayer | GuildPermissionRegisterPlayer | GuildPermissionGrantDelete | GuildPermissionDelete
)

