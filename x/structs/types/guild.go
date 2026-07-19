package types

import (
	"strings"

	"cosmossdk.io/math"
	//sdk "github.com/cosmos/cosmos-sdk/types"
)

// GuildBankDenomPrefix is the base-denom prefix for guild tokens: the full
// denom is "uguild.{guildId}". Kept in sync with GuildCache.GetBankDenom.
const GuildBankDenomPrefix = "uguild"

// ParseGuildBankDenom extracts the guild id from a guild bank token denom
// ("uguild.{guildId}"). It enforces the exact prefix so an arbitrary
// "foo.{guildId}" denom cannot be routed to a guild bank operation.
func ParseGuildBankDenom(denom string) (string, error) {
	denomSlice := strings.Split(denom, ".")
	if len(denomSlice) != 2 || denomSlice[0] != GuildBankDenomPrefix || denomSlice[1] == "" {
		return "", NewParameterValidationError("denom", 0, "invalid_format")
	}
	return denomSlice[1], nil
}

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
        BankConvertInFee: math.LegacyZeroDec(),
        BankConvertOutFee: math.LegacyZeroDec(),
	}
}


