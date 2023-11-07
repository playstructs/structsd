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
	player.PrimaryAddress = creator

	return nil
}

func (player *Player) SetPrimaryAddress(primaryAddress string) error {
	player.PrimaryAddress = primaryAddress

	return nil
}


func (player *Player) SetGuild(guildId uint64) error {

	player.GuildId = guildId

	return nil
}

func (player *Player) SetSquad(squadId uint64) error {

	player.SquadId = squadId

	return nil
}

func (player *Player) SetPlanetId(planetId uint64) error {

	player.PlanetId = planetId

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

