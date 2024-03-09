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

func (guild *Guild) SetEntrySubstationId(substationId string) error {

	guild.EntrySubstationId = substationId

	return nil
}


func (guild *Guild) SetPrimaryReactorId(reactorId string) error {

	guild.PrimaryReactorId = reactorId

	return nil
}


func (guild *Guild) SetOwner(playerId string) error {

	guild.Owner = playerId

	return nil
}


func (guild *Guild) SetJoinInfusionMinimum(joinInfusionMinimum uint64) error {

	guild.JoinInfusionMinimum = joinInfusionMinimum

	return nil
}


func CreateEmptyGuild() Guild {
	return Guild{
		Endpoint: "",
		Creator:  "",
		Owner: "",
        JoinInfusionMinimum: 0,
        JoinInfusionMinimumBypassByInvite: GuildJoinBypassLevel_closed,
        JoinInfusionMinimumBypassByRequest: GuildJoinBypassLevel_closed,
        PrimaryReactorId: "",
        EntrySubstationId: "",
	}
}


