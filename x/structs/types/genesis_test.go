package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
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
				ReactorList: []types.Reactor{
					{
						Id: 0,
					},
					{
						Id: 1,
					},
				},
				ReactorCount: 2,
				SubstationList: []types.Substation{
					{
						Id: 0,
					},
					{
						Id: 1,
					},
				},
				SubstationCount: 2,
				AllocationList: []types.Allocation{
					{
						Id: 0,
					},
					{
						Id: 1,
					},
				},
				AllocationCount: 2,
				AllocationProposalList: []types.AllocationProposal{
					{
						Id: 0,
					},
					{
						Id: 1,
					},
				},
				AllocationProposalCount: 2,
				GuildList: []types.Guild{
					{
						Id: 0,
					},
					{
						Id: 1,
					},
				},
				GuildCount: 2,
				PlayerList: []types.Player{
					{
						Id: 0,
					},
					{
						Id: 1,
					},
				},
				PlayerCount: 2,
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated reactor",
			genState: &types.GenesisState{
				ReactorList: []types.Reactor{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid reactor count",
			genState: &types.GenesisState{
				ReactorList: []types.Reactor{
					{
						Id: 1,
					},
				},
				ReactorCount: 0,
			},
			valid: false,
		},
		{
			desc: "duplicated substation",
			genState: &types.GenesisState{
				SubstationList: []types.Substation{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid substation count",
			genState: &types.GenesisState{
				SubstationList: []types.Substation{
					{
						Id: 1,
					},
				},
				SubstationCount: 0,
			},
			valid: false,
		},
		{
			desc: "duplicated allocation",
			genState: &types.GenesisState{
				AllocationList: []types.Allocation{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid allocation count",
			genState: &types.GenesisState{
				AllocationList: []types.Allocation{
					{
						Id: 1,
					},
				},
				AllocationCount: 0,
			},
			valid: false,
		},
		{
			desc: "duplicated allocationProposal",
			genState: &types.GenesisState{
				AllocationProposalList: []types.AllocationProposal{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid allocationProposal count",
			genState: &types.GenesisState{
				AllocationProposalList: []types.AllocationProposal{
					{
						Id: 1,
					},
				},
				AllocationProposalCount: 0,
			},
			valid: false,
		},
		{
			desc: "duplicated guild",
			genState: &types.GenesisState{
				GuildList: []types.Guild{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid guild count",
			genState: &types.GenesisState{
				GuildList: []types.Guild{
					{
						Id: 1,
					},
				},
				GuildCount: 0,
			},
			valid: false,
		},
		{
			desc: "duplicated player",
			genState: &types.GenesisState{
				PlayerList: []types.Player{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid player count",
			genState: &types.GenesisState{
				PlayerList: []types.Player{
					{
						Id: 1,
					},
				},
				PlayerCount: 0,
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	} {
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
