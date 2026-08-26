package v0_22_0

// UpgradeName is the on-chain upgrade plan name for the v0.22.0 binary.
//
// Consensus / handler changes (binary):
//
//   - The two remaining ante decorators keyed off the free-gas classification
//     now gate on message content instead, finishing the separation started in
//     v0.21.0 for StructsDecorator and ThrottleDecorator.
//
//     IsFreeTransaction requires EVERY message in a transaction to be a Structs
//     message, and IsFreeStakingTransaction requires every message to be a
//     staking message, so pairing one gameplay or staking message with any other
//     message left the transaction outside both rate limits. Paying for that
//     privilege cost nothing: chain.json sets fixed_min_gas_price to 0 and the
//     conditional mempool fee check returns early on a zero minimum.
//
//     CheckTxThrottleDecorator now runs whenever a transaction carries a gated
//     Structs message or a staking message, so a mixed transaction can no longer
//     buy extra per-address CheckTx admissions. It also constrains its
//     creatorGetter assertion to Structs messages, which was safe unguarded only
//     while it saw pure-Structs transactions.
//
//     StakingThrottleDecorator now runs whenever a transaction carries a staking
//     message, and skips messages from other modules rather than rejecting them.
//     One staking operation per address per block was already the rule on the
//     free path; a mixed transaction previously reserved nothing and could be
//     repeated without limit, firing the reactor and grid hooks behind each
//     delegation every time.
//
//     Transactions that mix a staking message with anything else are therefore
//     now subject to the same one-per-address-per-block limit as pure staking
//     transactions. That is the behaviour change operators should expect.
//
//   - The address registration proof is now domain-separated, chain-bound and
//     single-use. It used to be signed over "PLAYER<id>ADDRESS<addr>", a pure
//     function of two values that never change, which made it a permanent bearer
//     credential rather than a one-time consent. The only thing stopping a
//     second use was the address index, and AddressRevoke deletes exactly that,
//     so the player side could revoke an address, wait for it to receive funds,
//     resubmit the original proof and sweep the balance again with no fresh
//     consent from the key that owns it. Nothing bound the chain either, so a
//     proof verified on any other Structs chain carrying the same player id.
//
//     The proof is now built by types.AddressRegisterProofInput over the chain
//     id, a domain separator, the player id, the address, and a per-address
//     nonce read from state and burned on success. The nonce lives in its own
//     Address/nonce/ store and is deliberately NOT cleared by AddressRevoke,
//     which is what closes the replay; it is exported and imported in genesis
//     for the same reason.
//
//     This changes the bytes every client signs. Wallets and tooling that
//     produce address registration proofs must be updated in step with the
//     upgrade; the current nonce is served by the existing
//     /structs/address/{address} query as proofNonce, and is correct for
//     unassociated addresses too, which is the re-registration case.
//
// This upgrade is binary-only. There is no state migration: no persisted state
// is read or written by this upgrade and there are no store-key changes. The
// address nonce store starts empty and every address correctly begins at 0,
// because the sign-byte change invalidates every previously issued proof
// regardless. It
// exists so validators adopt the new ante behaviour at a coordinated height —
// nodes running a mix of the old and new binaries would accept different
// transactions and diverge.
const UpgradeName = "v0.22.0"
