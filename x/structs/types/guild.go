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

func (guild *Guild) SetEntrySubstationId(substationId uint64) error {

	guild.EntrySubstationId = substationId

	return nil
}


func (guild *Guild) SetPrimaryReactorId(reactorId uint64) error {

	guild.PrimaryReactorId = reactorId

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
		Owner: 0,
	    GuildJoinType: 0,
        InfusionJoinMinimum: 0,
        PrimaryReactorId: 0,
        EntrySubstationId: 0,
	}
}



type GuildPermission uint64

const (
    // 1
	GuildPermissionUpdate GuildPermission = 1 << iota
	// 2
	GuildPermissionRegisterPlayer
	// 4
	GuildPermissionDelete
)
const (
    GuildPermissionless GuildPermission = 0 << iota
	GuildPermissionAll = GuildPermissionUpdate | GuildPermissionRegisterPlayer | GuildPermissionDelete
)

