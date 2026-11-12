package app_test

import (
	"math/rand"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/group"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Seed proposals through the SDK keeper to represent proposals already stored
// before the upgrade. All attempted execution goes through signed FinalizeBlock.
func TestSignedGroupAuthzStructsBoundary(t *testing.T) {
	for _, path := range []string{"submit", "submit_try", "exec", "vote_try", "authz_exec", "native_control"} {
		t.Run(path, func(t *testing.T) {
			bApp, ctx := setupReactorAppGenesis(t, 1, true)
			key := secp256k1.GenPrivKey()
			address := sdk.AccAddress(key.PubKey().Address())
			player := appendAppPlayer(t, bApp.StructsKeeper, ctx, address)
			station := types.Substation{Id: "4-990", Owner: player.Id}
			bApp.StructsKeeper.SetSubstation(ctx, station)
			maximumID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacityMaximum, station.Id)
			pack := func(m sdk.Msg) *codectypes.Any {
				a, e := codectypes.NewAnyWithValue(m)
				require.NoError(t, e)
				return a
			}
			decision, e := codectypes.NewAnyWithValue(group.NewThresholdDecisionPolicy("1", time.Hour, 0))
			require.NoError(t, e)
			g, e := bApp.GroupKeeper.CreateGroupWithPolicy(ctx, &group.MsgCreateGroupWithPolicy{Admin: address.String(), Members: []group.MemberRequest{{Address: address.String(), Weight: "1"}}, DecisionPolicy: decision, GroupPolicyAsAdmin: true})
			require.NoError(t, e)
			inner := &types.MsgSubstationUpdateConnectionCapacityMaximum{Creator: address.String(), SubstationId: station.Id, ConnectionCapacityMaximum: 123}
			require.NoError(t, bApp.AuthzKeeper.SaveGrant(ctx, sdk.MustAccAddressFromBech32(g.GroupPolicyAddress), address, authz.NewGenericAuthorization(sdk.MsgTypeURL(inner)), nil))
			payload := &authz.MsgExec{Grantee: g.GroupPolicyAddress, Msgs: []*codectypes.Any{pack(inner)}}
			proposal := &group.MsgSubmitProposal{GroupPolicyAddress: g.GroupPolicyAddress, Proposers: []string{address.String()}, Messages: []*codectypes.Any{pack(payload)}}
			// A harmless group-policy metadata update is the non-Structs control.
			if path == "native_control" {
				proposal.Messages = []*codectypes.Any{pack(&group.MsgUpdateGroupPolicyMetadata{Admin: g.GroupPolicyAddress, GroupPolicyAddress: g.GroupPolicyAddress, Metadata: "updated"})}
			}
			var message sdk.Msg
			switch path {
			case "submit":
				message = proposal
			case "submit_try":
				proposal.Exec = group.Exec_EXEC_TRY
				message = proposal
			default:
				p, e := bApp.GroupKeeper.SubmitProposal(ctx, proposal)
				require.NoError(t, e)
				if path == "vote_try" {
					message = &group.MsgVote{ProposalId: p.ProposalId, Voter: address.String(), Option: group.VOTE_OPTION_YES, Exec: group.Exec_EXEC_TRY}
				} else {
					_, e = bApp.GroupKeeper.Vote(ctx, &group.MsgVote{ProposalId: p.ProposalId, Voter: address.String(), Option: group.VOTE_OPTION_YES})
					require.NoError(t, e)
					message = &group.MsgExec{ProposalId: p.ProposalId, Executor: address.String()}
					if path == "authz_exec" {
						message = &authz.MsgExec{Grantee: address.String(), Msgs: []*codectypes.Any{pack(message)}}
					}
				}
			}
			signer := bApp.AccountKeeper.GetAccount(ctx, address)
			config := authtx.NewTxConfig(bApp.AppCodec(), authtx.DefaultSignModes)
			tx, e := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), config, []sdk.Msg{message}, nil, ante.DefaultFreeGasCap, jailGateChainID, []uint64{signer.GetAccountNumber()}, []uint64{signer.GetSequence()}, key)
			require.NoError(t, e)
			wire, e := config.TxEncoder()(tx)
			require.NoError(t, e)
			block, e := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime(), Txs: [][]byte{wire}})
			require.NoError(t, e)
			result := block.TxResults[0]
			t.Logf("path=%s code=%d maximum=%d", path, result.Code, bApp.StructsKeeper.GetGridAttribute(bApp.NewContext(false), maximumID))
			if path == "native_control" {
				require.Zero(t, result.Code, result.Log)
				info, err := bApp.GroupKeeper.GroupPolicyInfo(bApp.NewContext(false), &group.QueryGroupPolicyInfoRequest{Address: g.GroupPolicyAddress})
				require.NoError(t, err)
				require.Equal(t, "updated", info.Info.Metadata)
			} else {
				require.Equal(t, ante.ErrNestedStructsMessage.ABCICode(), result.Code, result.Log)
			}
			require.Zero(t, bApp.StructsKeeper.GetGridAttribute(bApp.NewContext(false), maximumID))
		})
	}
}
