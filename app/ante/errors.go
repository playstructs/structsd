package ante

import (
	sdkerrors "cosmossdk.io/errors"
)

// AnteCodespace is the SDK error codespace for every typed error returned by
// the structsd ante chain. Clients can switch on
// (err.Codespace, err.ABCICode) to drive deterministic UX (auto-fix, refresh
// sequence, prompt user) instead of string-matching error messages.
const AnteCodespace = "structs-ante"

// Sentinel errors. Codes are deliberately stable so that client SDKs and
// downstream observability can correlate ABCI logs to known failure modes.
//
// Code range allocation:
//
//	2000-2009: Tx-shape rejects (size, msg count, message type)
//	2010-2019: Permission / player-registration rejects
//	2020-2029: Charge / discharge / per-tx duplication rejects
//	2030-2039: Proof-of-work / object-throttle rejects
//	2040-2049: Per-block / per-address quota rejects
//	2050-2059: Fee / gas-router rejects
var (
	// 2000-2009: tx shape
	ErrTxTooLarge = sdkerrors.Register(AnteCodespace, 2000,
		"tx exceeds maximum free tx size")
	ErrMsgCountExceeded = sdkerrors.Register(AnteCodespace, 2001,
		"tx exceeds maximum message count")
	ErrUnknownStructsMessage = sdkerrors.Register(AnteCodespace, 2002,
		"unknown structs message type (update ante maps)")
	ErrMissingCreator = sdkerrors.Register(AnteCodespace, 2003,
		"structs message has no accessible creator/signer")
	ErrNestedStructsMessage = sdkerrors.Register(AnteCodespace, 2004,
		"structs messages may not be wrapped in another message; submit them directly")

	// 2010-2019: permission / registration
	ErrUnregisteredAddress = sdkerrors.Register(AnteCodespace, 2010,
		"address is not registered as a player")
	ErrMissingPermission = sdkerrors.Register(AnteCodespace, 2011,
		"address lacks required permission bits for this message")

	// 2020-2029: charge / per-tx dedup
	ErrDuplicateChargeInTx = sdkerrors.Register(AnteCodespace, 2020,
		"transaction contains multiple charge-consuming messages from the same player; only one charge action per player per tx is allowed")
	ErrChargeAlreadyUsedThisBlock = sdkerrors.Register(AnteCodespace, 2021,
		"player has already used their charge action this block")
	ErrPlayerDischargedThisBlock = sdkerrors.Register(AnteCodespace, 2022,
		"player has zero charge this block (already discharged)")

	// 2030-2039: proof / object throttle
	ErrDuplicateProofInTx = sdkerrors.Register(AnteCodespace, 2030,
		"transaction contains multiple proof messages for the same struct id")
	ErrProofAlreadyAttemptedThisBlock = sdkerrors.Register(AnteCodespace, 2031,
		"proof for this struct was already attempted this block")
	ErrDuplicateThrottleKeyInTx = sdkerrors.Register(AnteCodespace, 2032,
		"transaction contains multiple messages with the same throttle key (e.g. same fleet, same planet explore, same registration)")
	ErrObjectThrottledThisBlock = sdkerrors.Register(AnteCodespace, 2033,
		"object is throttled for this block")

	// 2040-2049: per-block / per-address quotas
	ErrPlayerMsgCapExceeded = sdkerrors.Register(AnteCodespace, 2040,
		"player exceeded per-block message cap")
	ErrCheckTxAddrCapExceeded = sdkerrors.Register(AnteCodespace, 2041,
		"address exceeded CheckTx free-tx cap for this block")
	ErrDuplicateStakingInTx = sdkerrors.Register(AnteCodespace, 2042,
		"transaction contains multiple free staking messages from the same address")
	ErrStakingAlreadySubmittedThisBlock = sdkerrors.Register(AnteCodespace, 2043,
		"address already submitted a free staking tx this block")

	// 2050-2059: fee / gas
	ErrFreeTxMustHaveZeroFee = sdkerrors.Register(AnteCodespace, 2050,
		"free transaction must declare zero fee")
)
