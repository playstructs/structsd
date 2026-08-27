package types_test

import (
	"testing"

	"structs/x/structs/types"

	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				PortId: types.PortID,
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			// A provider's duration minimum is what stops a capacity resize from
			// rescaling an agreement down to no duration at all, and
			// GenesisImportProvider assigns the whole record, skipping the
			// setter that floors it at 1.
			desc: "provider with a zero duration minimum is invalid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				ProviderList: []types.Provider{{
					Id:              "7-0",
					DurationMinimum: 0,
					DurationMaximum: 1000,
				}},
			},
			valid: false,
		},
		{
			desc: "provider with an inverted duration range is invalid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				ProviderList: []types.Provider{{
					Id:              "7-0",
					DurationMinimum: 500,
					DurationMaximum: 100,
				}},
			},
			valid: false,
		},
		{
			desc: "provider with a sane duration range is valid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				ProviderList: []types.Provider{{
					Id:              "7-0",
					DurationMinimum: 1,
					DurationMaximum: 1000,
				}},
			},
			valid: true,
		},
		{
			/* A fleet whose id does not parse cannot be cached, so CommitAll
			 * never writes it - the fleet vanished while the players and
			 * structs pointing at it imported normally.
			 */
			desc: "fleet with an unparseable id is invalid",
			genState: &types.GenesisState{
				PortId:    types.PortID,
				FleetList: []types.Fleet{{Id: "not-a-fleet"}},
			},
			valid: false,
		},
		{
			desc: "fleet with a well-formed id is valid",
			genState: &types.GenesisState{
				PortId:    types.PortID,
				FleetList: []types.Fleet{{Id: "9-7"}},
			},
			valid: true,
		},
		{
			desc: "guild with entry rank zero is invalid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildList: []types.Guild{{
					Id:                                 "5-0",
					EntryRank:                          0,
					JoinInfusionMinimumBypassByRequest: types.GuildJoinBypassLevel_closed,
					JoinInfusionMinimumBypassByInvite:  types.GuildJoinBypassLevel_closed,
				}},
			},
			valid: false,
		},
		{
			desc: "guild with declared bypass levels is valid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildList: []types.Guild{{
					Id:                                 "5-0",
					EntryRank:                          types.DefaultEntryRank,
					JoinInfusionMinimumBypassByRequest: types.GuildJoinBypassLevel_member,
					JoinInfusionMinimumBypassByInvite:  types.GuildJoinBypassLevel_permissioned,
				}},
			},
			valid: true,
		},
		{
			desc: "guild with a name invalid under the pinned unicode rules is invalid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildList: []types.Guild{{
					Id:                                 "5-0",
					EntryRank:                          types.DefaultEntryRank,
					Name:                               "no", // below the 3-character minimum
					JoinInfusionMinimumBypassByRequest: types.GuildJoinBypassLevel_closed,
					JoinInfusionMinimumBypassByInvite:  types.GuildJoinBypassLevel_closed,
				}},
			},
			valid: false,
		},
		{
			desc: "guild with a valid name is valid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildList: []types.Guild{{
					Id:                                 "5-0",
					EntryRank:                          types.DefaultEntryRank,
					Name:                               "Alpha Guild",
					JoinInfusionMinimumBypassByRequest: types.GuildJoinBypassLevel_closed,
					JoinInfusionMinimumBypassByInvite:  types.GuildJoinBypassLevel_closed,
				}},
			},
			valid: true,
		},
		{
			// GenesisImportGuild assigns the record wholesale, so a genesis file
			// is the one path that skips GuildCache.SetJoinInfusionMinimumBypassBy*.
			desc: "guild with an undeclared byRequest bypass level is invalid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildList: []types.Guild{{
					Id:                                 "5-0",
					JoinInfusionMinimumBypassByRequest: types.GuildJoinBypassLevel(500),
				}},
			},
			valid: false,
		},
		{
			desc: "guild with an undeclared byInvite bypass level is invalid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildList: []types.Guild{{
					Id:                                "5-0",
					JoinInfusionMinimumBypassByInvite: types.GuildJoinBypassLevel(-1),
				}},
			},
			valid: false,
		},
		{
			desc: "membership application with declared join type and status is valid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildMembershipApplicationList: []types.GuildMembershipApplication{{
					GuildId:            "5-0",
					PlayerId:           "1-1",
					JoinType:           types.GuildJoinType_request,
					RegistrationStatus: types.RegistrationStatus_proposed,
				}},
			},
			valid: true,
		},
		{
			// The join type decides which side's consent an application stands
			// for, so an undeclared one is a record matching neither the request
			// nor the invite leg. GenesisImportGuildMembershipApplication assigns
			// the record wholesale and passes no setter that could have checked.
			desc: "membership application with an undeclared join type is invalid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildMembershipApplicationList: []types.GuildMembershipApplication{{
					GuildId:  "5-0",
					PlayerId: "1-1",
					JoinType: types.GuildJoinType(500),
				}},
			},
			valid: false,
		},
		{
			desc: "membership application with an undeclared registration status is invalid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildMembershipApplicationList: []types.GuildMembershipApplication{{
					GuildId:            "5-0",
					PlayerId:           "1-1",
					RegistrationStatus: types.RegistrationStatus(-1),
				}},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			// Every fixture above is about a list rather than about params, and a
			// zero-value Params is now invalid on its own account: the guild
			// charter difficulty range cannot be below 2, since CalculateDifficulty
			// divides by log10 of it. Filling in the defaults keeps each case
			// failing for the reason it declares.
			if tc.genState.Params.GuildCharterDifficultyRange == 0 {
				tc.genState.Params = types.DefaultParams()
			}

			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
