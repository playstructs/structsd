package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

func (squad *Squad) SetCreator(creator string) error {

	squad.Creator = creator

	return nil
}

func (squad *Squad) SetEntrySubstationId(substationId uint64) error {

	squad.EntrySubstationId = substationId

	return nil
}


func (squad *Squad) SetGuildId(guildId uint64) error {

	squad.GuildId = guildId

	return nil
}


func (squad *Squad) SetLeader(playerId uint64) error {

	squad.Leader = playerId

	return nil
}


func (squad *Squad) SetSquadJoinType(squadJoinType uint64) error {

	squad.SquadJoinType = squadJoinType

	return nil
}



func CreateEmptySquad() Squad {
	return Squad{
		Creator:  "",
		Leader: 0,
		GuildId: 0,
	    SquadJoinType: 0,
        EntrySubstationId: 0,
	}
}



type SquadPermission uint64

const (
    // 1
	SquadPermissionDelete SquadPermission = 1 << iota
	// 2
	SquadPermissionRegisterPlayer
	// 4
	SquadPermissionUpdateLeader
	// 8
	SquadPermissionUpdateEntrySubstation
	// 16
	SquadPermissionUpdateJoinType

)
const (
    SquadPermissionless SquadPermission = 0 << iota
	SquadPermissionAll = SquadPermissionDelete | SquadPermissionRegisterPlayer | SquadPermissionUpdateLeader | SquadPermissionUpdateEntrySubstation | SquadPermissionUpdateJoinType
)

