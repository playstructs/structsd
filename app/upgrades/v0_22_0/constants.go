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
//   - GuildMembershipJoin rejects a repeated infusion id instead of counting it
//     twice. msg.InfusionId is an unconstrained repeated field, and the branch
//     taken when an infusion's reactor already sits inside the destination guild
//     only reads the record and adds its fuel - it mutates nothing, so nothing
//     stopped the same infusion being named N times for N times its fuel.
//     JoinInfusionMinimum is the sole economic gate on a direct join, so that
//     bought entry to a guild for a fraction of the stake it advertises. Fuel
//     accumulation is also checked for overflow now, because the total is
//     compared against a minimum and a wrap produces a small number, which is
//     the direction that lets somebody in.
//
//   - GuildMembershipJoin now demands PermTokenMigrate before redelegating the
//     stake behind an infusion whose reactor sits outside the destination guild.
//     MsgGuildMembershipJoin names only its creator as a signer, so the infusion
//     address never authorizes that move and the handler was the whole
//     authorization - and it asked only for PermGuildMembership, which is a
//     membership permission, not a token one. ReactorBeginMigration gates the
//     identical staking call behind PermTokenMigrate.
//
//     Two callers gain nothing they should have had: an associated address
//     carrying only PermGuildMembership could move its own player's stake, and a
//     player granted PermGuildMembership over somebody else could move theirs.
//     No coins leave the delegator address, but the stake changes validator,
//     takes the redelegation lock and inherits the new validator's commission and
//     slashing exposure.
//
//     The bit stays out of PermissionMap because a join only migrates when an
//     infusion's reactor is elsewhere; demanding it of every join would break the
//     ordinary case that moves nothing.
//
//   - GuildMembershipJoinProxy now demands PermGuildMembership of an address that
//     is already registered to a player, while leaving genuinely unregistered
//     addresses permissionless.
//
//     UpsertPlayer is an in-get rather than an insert, so for a registered
//     address it returns that entire existing player and the handler goes on to
//     set its guild, rank, substation connection and profile. The direct join
//     path demands PermGuildMembership of the acting address before any of that,
//     and restricted secondary addresses exist so a low-trust key can play
//     without being able to move the player between guilds. Treating key
//     possession as sufficient let such a key do through the proxy exactly what
//     it is barred from doing directly.
//
//     The check mirrors PermissionCheck's Layer 1 rather than calling it: the
//     acting identity on the context is the proxy (msg.Creator), so a
//     PermissionCheck would test the wrong address's bits. The address that
//     signed the proof is the one consenting, so it is the one that must hold
//     the bit. A primary address holds PermAll, so ordinary players are
//     unaffected.
//
//   - GuildMembershipKick now clears the kicked player's direct permission row on
//     the guild. Kick previously cleared only GuildId and the rank, and
//     PermissionCheck reads a direct object grant with no membership predicate -
//     only the rank-derived branch is gated on being in a guild. So a dismissed
//     administrator kept whatever the guild had granted them directly, and
//     PermAdmin is the whole of what GuildUpdateOwnerId requires, which checks no
//     membership either: the dismissal handed the guild over.
//
//     The whole row is cleared rather than one bit, because it is keyed
//     (guild, player) and holds nothing but what that guild granted that player.
//     The owner's row is left alone - SetOwner stores ownership as PermGuildAll,
//     and ownership survives leaving because guilds are property and an owner
//     need not be a member. The kick loader already refuses to kick the owner, so
//     the exclusion is a backstop.
//
//     Scoped to kick, which is the guild explicitly withdrawing standing. A
//     player who leaves by joining another guild still keeps a direct grant on
//     the old one; the guild can withdraw it with PermissionRevokeOnObject.
//
//   - The guild join bypass level `member` no longer waives the signing key's own
//     permission ceiling. PermissionCheck has two independent layers - the key
//     that signed must hold the permission, and the player must have standing on
//     the object - and the `member` tier is a statement about the second only.
//     It was implemented by skipping PermissionCheck entirely, which dropped the
//     first as well, so a member's delegated key that carried no
//     PermGuildMembership could still approve, deny, invite and revoke.
//
//     The tier next door kept the ceiling: `permissioned` goes through
//     PermissionCheck. The same restricted key therefore got two different
//     answers depending on a guild setting that has nothing to do with it, and
//     opening a guild's join policy silently un-scoped every delegated key its
//     members held. Both CanInviteMembers and CanApproveMembershipRequest shared
//     the helper and both are fixed.
//
//     The check is CurrentContext.SignerPermissionCheck, which is Layer 1 on its
//     own. It cannot move to the ante: these messages are in
//     DynamicPermissionMessages, and a message in both maps gets no ante check at
//     all. Standing is still waived, so a member holding no grant on the guild
//     object continues to act on membership exactly as the tier intends.
//
//   - PermissionSetOnAddress and PermissionRevokeOnAddress refuse a write that
//     would leave a player's current primary address holding less than PermAll.
//
//     Authorizing the bits being destroyed (v0.21.0) closed the attack but not
//     the footgun: a player's own PermAll key acting on its own primary address
//     passes every check precisely because it still holds the rights it is about
//     to destroy. The resulting state is unrecoverable rather than reduced -
//     PlayerUpdatePrimaryAddress needs PermAll to rotate away, and a grant can
//     only hand over bits the granting address itself holds, so a narrowed
//     primary can neither restore itself nor authorize anything else to.
//
//     Nothing legitimate is blocked. A player who wants a narrow everyday key
//     rotates the primary to the address they intend to keep whole first, then
//     narrows the old one, which is untouched by the guard.
//
//   - Proof-of-work difficulty is computed in integers. It was
//     "64 - int(math.Log10(age)/math.Log10(range)*63)",
//     and math.Log10 is assembly on some architectures and portable Go on
//     others; those disagree in the last bit, and the truncation to an integer
//     turns that into a whole leading hexadecimal zero. Validators could require
//     different proofs for the same block and the same object - one accepting a
//     hash the other rejects, which is an app hash split rather than a wrong
//     answer. The amd64/arm64 agreement that held was a coincidence of Go's
//     implementation, not a guarantee.
//
//     CalculateDifficulty now takes uint64 and finds the same exponent by
//     multiplication: e = max{k : range^k <= age^63}, which is the logarithm
//     identity stated without a logarithm. It feeds every proof handler -
//     struct build, ore mining, ore refining, planet raid and the guild charter.
//
//     GAME RULE CHANGE: the integer form is the exact floor, so it differs from
//     the old float wherever 63*log_range(age) landed exactly on an integer and
//     the float fell a hair short. Across the reachable space that is 13 points,
//     all of them exact powers of the range's base (range 125 at ages 5 and 25,
//     range 2187 at ages 3 and 243, and so on). At each, difficulty drops by one:
//     the correct value, and the one s390x already computed. Everything else is
//     unchanged.
//
//   - PlanetRaidComplete refuses a planet that is not active, and a completed
//     planet no longer carries or starts a raid clock.
//
//     Completing a planet leaves its Owner stamped and its record in the store:
//     PlanetExplore points the player at a new one and the old record stays
//     loadable, still naming them. Every defence that makes a live raid hard
//     hangs off the owner's current planet - RefreshRaidVulnerability is only
//     ever called on owner.GetPlanet() - so a historical planet's clock was
//     started once by an arriving raider and then never reset, ageing while the
//     owner came and went from a planet they actually occupied, until the puzzle
//     decayed to a single leading zero. IsDefenderCommandStructVulnerable
//     compounded it by asking whether the owner's fleet was on station, which
//     for an abandoned planet is a question about somewhere else.
//
//     Stored ore is a player attribute rather than a planet one, so the obsolete
//     planet paid out the victim's entire live balance.
//
//     An active planet is always the owner's current one, because PlanetExplore
//     will not move a player on until AttemptComplete succeeds and that fails
//     while the planet still holds ore, so the active check is the whole of the
//     condition. AttemptComplete now clears both raid clocks outright, and
//     SetLocationListStart will not begin one on an inactive planet.
//
//     FleetMove refuses a completed planet as a destination, so a fleet cannot
//     arrive at one at all. Only player-chosen destinations are gated: the
//     internal moves - PeaceDeal, a completed raid, MigrateToNewPlanet - send a
//     fleet to its owner's current planet, which is active by construction, and
//     refusing those would strand fleets rather than protect anyone.
//
//   - PlayerSend and ProviderWithdrawBalance apply the bank's own recipient and
//     denom policy before moving coins.
//
//     BaseSendKeeper.SendCoins is not the bank's policy layer: it validates the
//     coin structure, applies any registered send restriction, and moves
//     balances. The blocked-address set and the per-denom send-enabled flags
//     live in the bank's MsgServer, which a module calling SendCoins directly
//     never goes through. So the app could declare the fee collector, the
//     distribution account and both staking pools unreachable and these handlers
//     reached them anyway, and governance could freeze a denom and they still
//     moved it.
//
//     The staking pools are the sharp end. Coins arriving there with no matching
//     delegation leave the pool balance disagreeing with staking's recorded
//     tokens, which surfaces as a panic on the next export-and-restart rather
//     than as a failed transaction.
//
//     These are the only two destinations in the module a transaction chooses.
//     Every other SendCoins sends to an address the module derived itself - a
//     provider pool, a guild bank, a player's own primary address - and those
//     are neither blocked nor attacker-chosen. PlayerSend additionally rejects a
//     malformed recipient rather than discarding the parse error and sending to
//     the empty address, and rejects a non-positive amount.
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
