package ante_test

import (
	"context"
	"fmt"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/group"
	"github.com/stretchr/testify/require"
	sante "structs/app/ante"
	"structs/x/structs/types"
)

type proposalReader map[uint64][]*codectypes.Any

func (p proposalReader) Proposal(_ context.Context, r *group.QueryProposalRequest) (*group.QueryProposalResponse, error) {
	messages, ok := p[r.ProposalId]
	if !ok {
		return nil, fmt.Errorf("proposal missing")
	}
	return &group.QueryProposalResponse{Proposal: &group.Proposal{Messages: messages}}, nil
}

func TestNestedGroupBoundary(t *testing.T) {
	pack := func(m sdk.Msg) *codectypes.Any {
		a, e := codectypes.NewAnyWithValue(m)
		require.NoError(t, e)
		return a
	}
	gameplay := pack(&types.MsgFleetMove{Creator: "player"})
	bank := pack(&banktypes.MsgSend{FromAddress: "policy", ToAddress: "recipient"})
	auth := pack(&authz.MsgExec{Grantee: "policy", Msgs: []*codectypes.Any{gameplay}})
	proposals := proposalReader{1: {auth}, 2: {bank}, 3: {pack(&group.MsgExec{ProposalId: 3})}, 4: {pack(&group.MsgExec{ProposalId: 1})}}
	for _, tc := range []struct {
		name   string
		msg    sdk.Msg
		reject bool
	}{
		{"direct group gameplay", &group.MsgSubmitProposal{Messages: []*codectypes.Any{gameplay}}, true},
		{"authz group submission", &authz.MsgExec{Msgs: []*codectypes.Any{pack(&group.MsgSubmitProposal{Messages: []*codectypes.Any{auth}})}}, true},
		{"stored nested group", &group.MsgExec{ProposalId: 4}, true},
		{"stored cycle", &group.MsgExec{ProposalId: 3}, true},
		{"missing proposal", &group.MsgExec{ProposalId: 99}, true},
		{"vote executes", &group.MsgVote{ProposalId: 1, Exec: group.Exec_EXEC_TRY}, true},
		{"vote only", &group.MsgVote{ProposalId: 1}, false},
		{"withdraw", &group.MsgWithdrawProposal{ProposalId: 1}, false},
		{"bank execution", &group.MsgExec{ProposalId: 2}, false},
		{"bank submission", &group.MsgSubmitProposal{Messages: []*codectypes.Any{bank}}, false},
		{"bank authz", &authz.MsgExec{Msgs: []*codectypes.Any{bank}}, false},
		{"governance params", &govtypes.MsgSubmitProposal{Messages: []*codectypes.Any{pack(&types.MsgUpdateParams{})}}, false},
		{"malformed group", &authz.MsgExec{Msgs: []*codectypes.Any{{TypeUrl: sdk.MsgTypeURL(&group.MsgExec{}), Value: []byte{255}}}}, true},
		{"nil payload", &group.MsgSubmitProposal{Messages: []*codectypes.Any{nil}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range []string{"deliver", "check", "recheck", "simulate"} {
				ctx := newTestCtx()
				if mode == "check" {
					ctx = ctx.WithIsCheckTx(true)
				}
				if mode == "recheck" {
					ctx = ctx.WithIsCheckTx(true).WithIsReCheckTx(true)
				}
				next, called := identityHandler()
				_, err := sante.NewNestedStructsMsgDecorator(proposals).AnteHandle(ctx, mockTx{msgs: []sdk.Msg{tc.msg}}, mode == "simulate", next)
				if tc.reject {
					require.ErrorIs(t, err, sante.ErrNestedStructsMessage, mode)
					require.False(t, *called)
				} else {
					require.NoError(t, err, mode)
					require.True(t, *called)
				}
			}
		})
	}
}

func TestNestedGroupTraversalBounds(t *testing.T) {
	pack := func(m sdk.Msg) *codectypes.Any {
		a, e := codectypes.NewAnyWithValue(m)
		require.NoError(t, e)
		return a
	}
	bank := pack(&banktypes.MsgSend{})
	// One root wrapper plus 255 leaves fits the transaction-wide visit budget.
	for _, count := range []int{255, 256} {
		messages := make([]*codectypes.Any, count)
		for i := range messages {
			messages[i] = bank
		}
		next, _ := identityHandler()
		_, err := sante.NewNestedStructsMsgDecorator(nil).AnteHandle(newTestCtx(), mockTx{msgs: []sdk.Msg{&authz.MsgExec{Msgs: messages}}}, false, next)
		if count == 255 {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, sante.ErrNestedStructsMessage)
		}
	}
	// Mixed group/authz wrappers share the same depth limit, even for bank payloads.
	var msg sdk.Msg = &authz.MsgExec{Msgs: []*codectypes.Any{bank}}
	for i := 0; i < 9; i++ {
		msg = &group.MsgSubmitProposal{Messages: []*codectypes.Any{pack(msg)}}
	}
	next, _ := identityHandler()
	_, err := sante.NewNestedStructsMsgDecorator(nil).AnteHandle(newTestCtx(), mockTx{msgs: []sdk.Msg{msg}}, false, next)
	require.ErrorIs(t, err, sante.ErrNestedStructsMessage)
}

type countingProposalReader struct {
	proposalReader
	reads map[uint64]int
}

func (c countingProposalReader) Proposal(ctx context.Context, r *group.QueryProposalRequest) (*group.QueryProposalResponse, error) {
	c.reads[r.ProposalId]++
	return c.proposalReader.Proposal(ctx, r)
}

func TestNestedGroupReadsEachProposalOnce(t *testing.T) {
	pack := func(m sdk.Msg) *codectypes.Any {
		a, e := codectypes.NewAnyWithValue(m)
		require.NoError(t, e)
		return a
	}
	wrap := func(m sdk.Msg, levels int) *codectypes.Any {
		a := pack(m)
		for i := 0; i < levels; i++ {
			a = pack(&authz.MsgExec{Msgs: []*codectypes.Any{a}})
		}
		return a
	}
	bank := pack(&banktypes.MsgSend{})
	exec1, exec2, exec3 := &group.MsgExec{ProposalId: 1}, &group.MsgExec{ProposalId: 2}, &group.MsgExec{ProposalId: 3}
	proposals := proposalReader{1: {bank}, 2: {wrap(&banktypes.MsgSend{}, 1)}, 3: {pack(exec2)}}
	run := func(msgs ...*codectypes.Any) (map[uint64]int, error) {
		reader := countingProposalReader{proposals, map[uint64]int{}}
		next, _ := identityHandler()
		_, err := sante.NewNestedStructsMsgDecorator(reader).AnteHandle(newTestCtx(), mockTx{msgs: []sdk.Msg{&authz.MsgExec{Msgs: msgs}}}, false, next)
		return reader.reads, err
	}

	// Each repeated reference still spends what a rescan would: the root wrapper,
	// then the MsgExec and its one stored leaf per reference.
	for _, refs := range []int{127, 128} {
		msgs := make([]*codectypes.Any, refs)
		for i := range msgs {
			msgs[i] = pack(exec1)
		}
		reads, err := run(msgs...)
		require.Equal(t, map[uint64]int{1: 1}, reads)
		if refs == 127 {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, sante.ErrNestedStructsMessage)
		}
	}

	// A repeated reference reaches as deep as a rescan would, including through a
	// proposal whose own height was recorded while replaying another.
	for _, tc := range []struct {
		name   string
		msgs   []*codectypes.Any
		reads  map[uint64]int
		reject bool
	}{
		{"replay at depth limit", []*codectypes.Any{pack(exec2), wrap(exec2, 5)}, map[uint64]int{2: 1}, false},
		{"replay past depth limit", []*codectypes.Any{pack(exec2), wrap(exec2, 6)}, map[uint64]int{2: 1}, true},
		{"nested replay at depth limit", []*codectypes.Any{pack(exec2), pack(exec3), wrap(exec3, 4)}, map[uint64]int{2: 1, 3: 1}, false},
		{"nested replay past depth limit", []*codectypes.Any{pack(exec2), pack(exec3), wrap(exec3, 5)}, map[uint64]int{2: 1, 3: 1}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads, err := run(tc.msgs...)
			require.Equal(t, tc.reads, reads)
			if tc.reject {
				require.ErrorIs(t, err, sante.ErrNestedStructsMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
