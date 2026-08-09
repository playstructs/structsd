package ante

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// NestedStructsMsgDecorator rejects Structs messages smuggled inside an authz
// MsgExec.
//
// Every other Structs ante check walks tx.GetMsgs(), which reports the outer
// MsgExec and never the messages it carries. A grantee could therefore execute
// gameplay with none of the player registration, permission, charge or throttle
// gating applied, and with the nested creator set to the granter. The keeper
// handlers would still enforce their own permission checks, but the ante layer
// would be blind, so the per-block quotas and throttles that protect the free
// tier would not apply.
//
// There is no gameplay reason to wrap a Structs message this way -- scoped
// authority is already expressed by registering an address with a permission
// bitfield -- so this denies the shape outright rather than trying to gate
// nested messages correctly.
//
// Governance is deliberately not covered here: MsgSubmitProposal carrying
// MsgUpdateParams is the supported way to change module params, and gov already
// validates that a proposal's messages are signed by the governance authority,
// which no player-facing Structs message is.
type NestedStructsMsgDecorator struct{}

func NewNestedStructsMsgDecorator() NestedStructsMsgDecorator {
	return NestedStructsMsgDecorator{}
}

func (d NestedStructsMsgDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	for _, msg := range tx.GetMsgs() {
		exec, ok := msg.(*authz.MsgExec)
		if !ok {
			continue
		}

		// Read the type URL off the Any directly. Unpacking would need the
		// interface registry and would fail closed on unregistered types
		// anyway; the URL is the whole basis for the decision.
		for _, nested := range exec.Msgs {
			if nested == nil {
				continue
			}
			if IsStructsMessage(nested.TypeUrl) {
				return ctx, observeReject(ctx, "NestedStructsMsgDecorator",
					errorsmod.Wrapf(ErrNestedStructsMessage, "%s inside %s", nested.TypeUrl, sdk.MsgTypeURL(msg)))
			}
		}
	}

	return next(ctx, tx, simulate)
}
