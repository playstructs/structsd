package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

func (guild *Guild) SetCreator(creator string) error {

	guild.Creator = creator

	return nil
}

func (guild *Guild) SetEndpoint(endpoint string) error {

	guild.Endpoint = endpoint

	return nil
}

func (guild *Guild) SetOwner(playerId uint64) error {

	guild.Owner = playerId

	return nil
}

func CreateEmptyGuild() Guild {
	return Guild{
		Endpoint: "",
		Creator:  "",
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

