package ante

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/group"
)

// GroupProposalReader resolves stored payloads, including pre-upgrade proposals.
type GroupProposalReader interface {
	Proposal(context.Context, *group.QueryProposalRequest) (*group.QueryProposalResponse, error)
}

// NestedStructsMsgDecorator rejects gameplay dispatched without the Structs ante
// checks. Both authz and group route their inner messages directly to handlers.
// Inspect submissions even without EXEC_TRY, and resolve execution references at
// admission time rather than assuming all stored proposals were checked before.
// Governance remains authority-gated and supports MsgUpdateParams as before.
// This decorator reads state and must run AFTER signature verification and fee
// deduction, under the transaction gas meter.
type NestedStructsMsgDecorator struct{ proposals GroupProposalReader }

const (
	maxAuthzNestingDepth = 8
	// Bound the entire traversal, including repeated references and proposal cycles.
	maxNestedMessageVisits = 256
)

func NewNestedStructsMsgDecorator(proposals GroupProposalReader) NestedStructsMsgDecorator {
	return NestedStructsMsgDecorator{proposals: proposals}
}

func (d NestedStructsMsgDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	scan := nestedMessageScan{ctx: ctx, proposals: d.proposals}
	for _, msg := range tx.GetMsgs() {
		if err := scan.message(msg, 0); err != nil {
			return ctx, observeReject(ctx, "NestedStructsMsgDecorator", err)
		}
	}
	return next(ctx, tx, simulate)
}

type nestedMessageScan struct {
	ctx       sdk.Context
	proposals GroupProposalReader
	visits    int
	deepest   int
	// A repeated reference must pass or fail exactly as a rescan would, so it
	// replays the proposal's recorded height and visits instead of reading it again.
	scanned map[uint64]scannedProposal
}

type scannedProposal struct{ height, visits int }

func (s *nestedMessageScan) visit(depth int) error {
	s.visits++
	s.reach(depth)
	if depth > maxAuthzNestingDepth || s.visits > maxNestedMessageVisits {
		return errorsmod.Wrap(ErrNestedStructsMessage, "nested message traversal limit exceeded")
	}
	return nil
}

func (s *nestedMessageScan) reach(depth int) {
	if depth > s.deepest {
		s.deepest = depth
	}
}

func (s *nestedMessageScan) message(msg sdk.Msg, depth int) error {
	if err := s.visit(depth); err != nil {
		return err
	}
	if depth > 0 && IsStructsMessage(sdk.MsgTypeURL(msg)) {
		return errorsmod.Wrapf(ErrNestedStructsMessage, "nested %s", sdk.MsgTypeURL(msg))
	}
	switch m := msg.(type) {
	case *authz.MsgExec:
		return s.messages(m.Msgs, depth+1)
	case *group.MsgSubmitProposal:
		return s.messages(m.Messages, depth+1)
	case *group.MsgExec:
		return s.proposal(m.ProposalId, depth+1)
	case *group.MsgVote:
		if m.Exec == group.Exec_EXEC_TRY {
			return s.proposal(m.ProposalId, depth+1)
		}
	}
	return nil
}

func (s *nestedMessageScan) proposal(id uint64, depth int) error {
	if depth > maxAuthzNestingDepth || s.proposals == nil {
		return errorsmod.Wrap(ErrNestedStructsMessage, "cannot inspect group proposal")
	}
	if seen, ok := s.scanned[id]; ok {
		s.visits += seen.visits
		s.reach(depth + seen.height)
		if depth+seen.height > maxAuthzNestingDepth || s.visits > maxNestedMessageVisits {
			return errorsmod.Wrap(ErrNestedStructsMessage, "nested message traversal limit exceeded")
		}
		return nil
	}
	result, err := s.proposals.Proposal(s.ctx, &group.QueryProposalRequest{ProposalId: id})
	if err != nil {
		return errorsmod.Wrapf(ErrNestedStructsMessage, "cannot inspect group proposal %d: %v", id, err)
	}
	if result == nil || result.Proposal == nil {
		return errorsmod.Wrap(ErrNestedStructsMessage, "missing group proposal")
	}
	outerDeepest, outerVisits := s.deepest, s.visits
	s.deepest = depth
	if err := s.messages(result.Proposal.Messages, depth); err != nil {
		return err
	}
	if s.scanned == nil {
		s.scanned = make(map[uint64]scannedProposal)
	}
	s.scanned[id] = scannedProposal{height: s.deepest - depth, visits: s.visits - outerVisits}
	s.reach(outerDeepest)
	return nil
}

func (s *nestedMessageScan) messages(messages []*codectypes.Any, depth int) error {
	for _, nested := range messages {
		if nested == nil {
			return errorsmod.Wrap(ErrNestedStructsMessage, "nil nested message")
		}
		if IsStructsMessage(nested.TypeUrl) {
			return errorsmod.Wrapf(ErrNestedStructsMessage, "nested %s", nested.TypeUrl)
		}
		// Decode only executable containers. Authorization grants are data, not calls.
		var msg interface {
			sdk.Msg
			Unmarshal([]byte) error
		}
		switch nested.TypeUrl {
		case sdk.MsgTypeURL(&authz.MsgExec{}):
			msg = &authz.MsgExec{}
		case sdk.MsgTypeURL(&group.MsgSubmitProposal{}):
			msg = &group.MsgSubmitProposal{}
		case sdk.MsgTypeURL(&group.MsgExec{}):
			msg = &group.MsgExec{}
		case sdk.MsgTypeURL(&group.MsgVote{}):
			msg = &group.MsgVote{}
		default:
			if err := s.visit(depth); err != nil {
				return err
			}
			continue
		}
		if err := msg.Unmarshal(nested.Value); err != nil {
			return errorsmod.Wrapf(ErrNestedStructsMessage, "cannot decode nested %s: %v", nested.TypeUrl, err)
		}
		if err := s.message(msg, depth); err != nil {
			return err
		}
	}
	return nil
}
