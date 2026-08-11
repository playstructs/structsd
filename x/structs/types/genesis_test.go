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
			desc: "guild with declared bypass levels is valid",
			genState: &types.GenesisState{
				PortId: types.PortID,
				GuildList: []types.Guild{{
					Id:                                 "5-0",
					JoinInfusionMinimumBypassByRequest: types.GuildJoinBypassLevel_member,
					JoinInfusionMinimumBypassByInvite:  types.GuildJoinBypassLevel_permissioned,
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
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
