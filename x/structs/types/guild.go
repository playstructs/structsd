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


func (guild *Guild) SetJoinInfusionMinimum(joinInfusionMinimum uint64) error {

	guild.JoinInfusionMinimum = joinInfusionMinimum

	return nil
}

func (guild *Guild) SetJoinInfusionMinimumBypassByInvite(guildJoinBypassLevel uint64) error {

	guild.JoinInfusionMinimumBypassByInvite = guildJoinBypassLevel

	return nil
}

func (guild *Guild) SetJoinInfusionMinimumBypassByRequest(guildJoinBypassLevel uint64) error {

	guild.JoinInfusionMinimumBypassByRequest = guildJoinBypassLevel

	return nil
}

func CreateEmptyGuild() Guild {
	return Guild{
		Endpoint: "",
		Creator:  "",
		Owner: 0,
        JoinInfusionMinimum: 0,
        JoinInfusionMinimumBypassByInvite: GuildJoinBypassLevel_Closed,
        JoinInfusionMinimumBypassByRequest: GuildJoinBypassLevel_Closed,
        PrimaryReactorId: 0,
        EntrySubstationId: 0,
	}
}



type GuildPermission Permission

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

