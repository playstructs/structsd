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

// NormalizeBankFees replaces the nil LegacyDec values produced when records
// written before v0.21.0 are decoded. It reports whether the guild changed.
func (guild *Guild) NormalizeBankFees() bool {
	changed := false
	if guild.BankConvertInFee.IsNil() {
		guild.BankConvertInFee = math.LegacyZeroDec()
		changed = true
	}
	if guild.BankConvertOutFee.IsNil() {
		guild.BankConvertOutFee = math.LegacyZeroDec()
		changed = true
	}
	return changed
}

// IsValid reports whether level is one of the values declared in
// proto/structs/structs/keys.proto. proto3 enums are open — the generated
// decoder shifts bytes into an int32 without consulting the enum — so a stored
// or message-carried level is untrusted input until this says otherwise.
//
// The membership test reads the generated name map rather than listing the
// three constants, so a value added to the proto is accepted here the moment it
// is generated. Note what that does *not* buy: a new value still has no case in
// the guild_cache.go switches, and their default branches deny it. That pairing
// is deliberate — a level nobody has written policy for is storable but grants
// nothing.
func (level GuildJoinBypassLevel) IsValid() bool {
	_, declared := GuildJoinBypassLevel_name[int32(level)]
	return declared
}

// IsValid reports whether joinType is one of the values declared in
// proto/structs/structs/keys.proto. Same open-enum reasoning as
// GuildJoinBypassLevel.IsValid, and the same stakes: the join type selects which
// side's consent an application stands for, so an undeclared one reaches the
// approve path as a record matching neither the request nor the invite leg.
func (joinType GuildJoinType) IsValid() bool {
	_, declared := GuildJoinType_name[int32(joinType)]
	return declared
}

// IsValid reports whether status is one of the values declared in
// proto/structs/structs/keys.proto. An undeclared status is what
// GuildMembershipApplicationCache.requirePending refuses, so validating it on the
// way in keeps a genesis file from writing a row that can never be acted on.
func (status RegistrationStatus) IsValid() bool {
	_, declared := RegistrationStatus_name[int32(status)]
	return declared
}

// NormalizeJoinBypassLevels clamps either bypass field to closed when it holds
// a value outside the declared enum, which records written before the update
// handlers validated their input can. It reports whether the guild changed.
//
// Closed is the recoverable direction: CanUpdateJoinConstraintsBy reads only
// the permission bit and never the bypass level, so an owner reopens the guild
// with one transaction.
func (guild *Guild) NormalizeJoinBypassLevels() bool {
	changed := false
	if !guild.JoinInfusionMinimumBypassByRequest.IsValid() {
		guild.JoinInfusionMinimumBypassByRequest = GuildJoinBypassLevel_closed
		changed = true
	}
	if !guild.JoinInfusionMinimumBypassByInvite.IsValid() {
		guild.JoinInfusionMinimumBypassByInvite = GuildJoinBypassLevel_closed
		changed = true
	}
	return changed
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
		Endpoint:                           "",
		Creator:                            "",
		Owner:                              "",
		JoinInfusionMinimum:                0,
		JoinInfusionMinimumBypassByInvite:  GuildJoinBypassLevel_closed,
		JoinInfusionMinimumBypassByRequest: GuildJoinBypassLevel_closed,
		PrimaryReactorId:                   "",
		EntrySubstationId:                  "",
		BankConvertInFee:                   math.LegacyZeroDec(),
		BankConvertOutFee:                  math.LegacyZeroDec(),
	}
}
