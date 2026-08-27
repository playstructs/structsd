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
//   - PlayerUpdateGuildRank re-applies the signing key's permission ceiling
//     before falling back to rank authority.
//
//     PermissionCheck fails for two unrelated reasons: the key that signed does
//     not carry PermAdmin, or it does and the player has no standing on the
//     guild. The handler treated both alike and fell through to a comparison of
//     player-level ranks, so a deliberately restricted secondary key inherited
//     its player's whole rank authority and could re-rank every member below
//     them - including promoting a colluding member up to the actor's own rank,
//     which carries rank-derived grants on guild objects.
//
//     Rank substitutes for standing, not for the ceiling, so the fallback now
//     demands PermAdmin on the signing address alone (SignerPermissionCheck,
//     the same helper the guild bypass tiers use). A primary address holds
//     PermAll, so an ordinary member is unaffected.
//
//     This was the only rank check in the module that substituted for a
//     permission. GuildUpdateEntryRank and the membership kick path both apply
//     their rank bound after a PermissionCheck that has already enforced the
//     ceiling.
//
//   - ProviderCreate resolves its substation id instead of merely allocating a
//     cache for it.
//
//     cc.GetSubstation is a cache allocator: it wraps any string and reads
//     nothing, which is correct for an id taken off something already loaded
//     from state and wrong for one a transaction supplied. The permission check
//     that followed was not a backstop - object permissions are keyed by the raw
//     id string, and registration grants every player PermAll on their own
//     player id, so submitting that id passed CanAllocateAsSourceBy against a
//     substation that was never there.
//
//     Grid attribute ids are derived from the object id alone, so the resulting
//     provider read its available capacity from the player's own capacity and
//     load counters and sold grid the substation accounting knew nothing about.
//
//     cc.GetExistingSubstation checks both the id's type namespace and the
//     record's existence. The type check is not redundant: a substation stored
//     under a wrongly-typed id, which only a hand-written genesis could produce,
//     would collide with another object's grid attributes while existing
//     perfectly well.
//
//   - Every handler taking a provider id from a message resolves it through
//     cc.GetExistingProvider rather than allocating a cache for it. Eight of
//     them did the latter: the six provider update handlers, ProviderDelete and
//     AgreementOpen.
//
//     Object permissions are keyed by the raw id string with no type
//     namespacing, and registration grants every player PermAll on their own
//     player id, so submitting that id as a provider id collided with a record
//     that genuinely exists and CanBeUpdatedBy passed on a provider that did
//     not. The handler then mutated and committed a zero-valued provider.
//
//     That commit panics in the KV store on the empty Provider.Id and BaseApp
//     turns the panic into a failed transaction, so nothing was persisted -
//     which is why the reachable damage was limited to the attacker's own
//     transaction. It is not a check, though: the same shape wrote real state in
//     AllocationTransfer, where the write was keyed by something non-empty and
//     nothing tripped.
//
//     This is the third object type to get the treatment GetExistingPlayer
//     introduced, after players and substations, and the arch guard now covers
//     substations and providers together.
//
//   - StructGeneratorInfuse moves and burns only the coin it validated.
//
//     msg.InfuseAmount is a free-form string and ParseCoinsNormalized accepts a
//     comma-separated list, sorting it by denom. The handler checked
//     infusionAmount[0] and then passed the whole slice to
//     SendCoinsFromAccountToModule and BurnCoins, so "1ualpha,1000000uguild.1-2"
//     put a valid denom at index 0, passed, and destroyed the guild tokens.
//     Only the first coin was ever credited as fuel, so the rest burned for
//     nothing.
//
//     That crossed an authorization boundary as well as an accounting one. This
//     handler asks only for PermTokenInfuse and spends the player's primary
//     account, so a delegate scoped to infusion alone could destroy asset
//     classes PermGuildTokenBurn exists to protect - guild tokens are minted
//     straight into that same primary account.
//
//     The handler now requires exactly one coin, and rebuilds a canonical
//     single-denomination value after the alpha conversion rather than reusing
//     the parsed one: what moves has to be what was checked. The BurnCoins error
//     is propagated too, since by then the coins are already in the module
//     account.
//
//   - Deleting a substation disconnects an arriving providerAgreement allocation
//     instead of destroying it.
//
//     Ending an agreement early has two prices. AgreementClose charges the
//     consumer cancellation penalty; PrematureCloseByAllocation pays the
//     provider one to the consumer, which is right when the provider walks away
//     and a gift when the consumer does. Substation deletion destroyed every
//     inbound allocation, and an agreement's allocation can arrive at a
//     substation the consumer owns - so a consumer could choose the favourable
//     price by deleting a substation rather than closing the agreement.
//     AllocationDelete already refused to tear down a providerAgreement
//     allocation directly; this was the same thing one level up.
//
//     The agreement now survives with no destination, exactly as it does between
//     AgreementOpen and the first connect, and can only be ended through the two
//     paths that price it.
//
//     Disconnect rather than refuse the deletion, because
//     SubstationAllocationConnect asks for no rights on the destination: anyone
//     may point an allocation at anyone's substation, so refusing would let a
//     stranger's agreement pin a substation in place forever.
//
//     Outbound allocations are unchanged. An agreement is sourced from the
//     provider's substation, so deleting that one is the provider ending service
//     and the provider penalty it pays is the correct price.
//
//   - Reactor infusion fuel is converted from delegation shares by truncating
//     against a multiply-first quotient, matching Validator.TokensFromShares,
//     instead of rounding a divide-first one.
//
//     Fuel is written per delegation and LegacyDec.RoundInt is banker's
//     rounding, so a shard worth exactly x.5 rounded up. After a 1% slash a
//     50-share delegation is worth exactly 49.5, so splitting a million across
//     twenty thousand addresses returned every shard to 50 and left a million
//     Fuel - and a million of grid capacity - standing on 990,000 tokens.
//     Nothing sums Fuel against validator.Tokens, so nothing noticed. Truncation
//     bounds each shard by its true value, so the total is bounded by the stake
//     and the residue is dust nobody is credited for.
//
//     Dividing before multiplying rounded the ratio to LegacyDec's 18 places
//     before it met the pool, so that error scaled with the validator. The SDK
//     orders it the other way and this now agrees with the SDK, which is the
//     only answer that is actually what the stake is worth.
//
//   - A recovery message sent from a player's primary address no longer spends
//     the shared per-player message quota.
//
//     The quota is keyed by player, so every address of a player draws on one
//     budget, and StructsDecorator charges it in the ante - before any handler
//     runs, and therefore whether or not the message succeeds. The SDK commits
//     the ante cache and discards only the message cache, so a delegated key
//     could spend the whole block's budget on messages that all fail.
//
//     That is griefing everywhere except one place, where it is a trap: the
//     lockout covered MsgAddressRevoke, so the key doing the griefing blocked
//     the one message that would remove it. The rule it broke is already stated
//     for delegation transfers - a revoke is the response to a compromised key,
//     so anything that key can sustain must not be able to block it.
//
//     RecoveryMessages holds the exempt set, MsgAddressRevoke and
//     MsgPlayerUpdatePrimaryAddress, and the exemption is the primary address
//     rather than the message type: a delegate naming a recovery message gets
//     no exemption at all. Both already demand PermDelete or PermAll at the
//     ante and neither carries a throttle key, which is what makes them safe to
//     exempt. This does not stop a bad key exhausting the quota. It stops that
//     from being unrecoverable.
//
//   - GridCascade is bounded to GridCascadeBlockBudget units of work per block,
//     and the cascade queue is FIFO rather than keyed by object id.
//
//     The cascade ran to exhaustion in the EndBlocker, which has no gas meter,
//     over a graph an attacker sizes for nothing: an allocation of one power
//     feeds a substation, that substation allocates the same power onward, and
//     two free messages add a link. Destroying the root collapsed the whole
//     chain inside one block - a halt rather than a slow block, and a
//     deterministic one, so every retry of that block did the same thing.
//
//     The budget counts queue entries visited as well as allocations destroyed;
//     charging only for destroys would bound the shedding and leave the walk
//     unbounded. What the budget does not reach stays queued for the next block.
//
//     It also bounds the reads. Capping the destroys said nothing about how many
//     allocations were loaded to find them: GetAllAllocationBySource
//     materialized every allocation on a source - an index read, a store read
//     and a retained cache object each - before the loop could decide it had had
//     enough. A source's fan-out is attacker-sized in the same way its depth is,
//     capacity being splittable into one-power allocations, so a single wide
//     source put the unbounded work back exactly where the budget had removed
//     it. GetAllocationsBySourceUpTo bounds the iteration and reports whether
//     more remain, which is what lets the loop tell "this source cannot be
//     brought under capacity" - the Grid Queue problem warn - from "I stopped
//     early", which requeues.
//
//     FIFO is the other half and is a security property, not tidiness. Ordering
//     by object id sorts lexicographically, so an attacker could hold a
//     substation whose id sorts late out of the queue indefinitely by keeping
//     cheaper-sorting entries in front of it, and an object that is never
//     reached goes on powering structs it has no capacity for. Under a sequence
//     nothing can be inserted ahead of an entry already queued. Re-appending a
//     pending object is a no-op rather than a move to the back, for the same
//     reason.
//
//     A deferred object cannot sell what it does not have: every gate that
//     grants power compares before it subtracts - SubstationCache and
//     PlayerCache GetAvailableCapacity, CanSupportLoadAddition - so an
//     over-subscribed object reports zero headroom while it waits. What it keeps
//     doing is powering what is already attached, which is the "one last block
//     of power" this has always accepted, over more blocks.
//
//     GAME RULE CHANGE: a cascade larger than the budget now sheds over several
//     blocks instead of one. A grid a real guild builds is orders of magnitude
//     below the budget and still finishes in the block it starts.
//
//     The queue is also imported at genesis now. It was exported and never
//     imported, which was survivable only while the cascade drained it inside
//     the block that filled it; deferred work would otherwise be lost across a
//     restore, leaving those objects over-subscribed with nothing left to
//     schedule them.
//
//   - A capacity resize that rescales an agreement to zero duration is refused,
//     independently of the provider's published minimum.
//
//     The rescale trades duration for capacity and truncates, so a small,
//     nearly-expired agreement grown by a large amount lands on a duration of
//     zero, which sets EndBlock to the current block. That is not "already
//     expired": AgreementExpirations runs in the EndBlocker, after every message
//     in the block, and matches EndBlock == currentBlock exactly. The raised
//     capacity, provider load and allocation power are therefore live for the
//     rest of the block and usable by a later message in the same transaction,
//     while the collateral still covers only the old capacity.
//
//     AgreementDurationVerify already caught this on any chain a transaction
//     built, because every provider setter floors DurationMinimum at 1. That was
//     a provider policy standing in for a safety property, and one whole-record
//     write away from being absent: GenesisImportProvider assigns a Provider
//     straight onto the cache and passes no setter. rescaledDuration now refuses
//     zero on its own account, which covers a record already on disk, and
//     GenesisState.Validate rejects a provider whose duration minimum is zero or
//     whose range is inverted, which stops new ones. The pairing is the same one
//     used for guild join bypass levels.
//
//   - Genesis import restores a provider's checkpoint block, and stamps the
//     genesis height on a provider that arrives without one.
//
//     checkpointBlock is a clock and is the one grid attribute the import cannot
//     derive from the objects it rebuilds. Load and capacity are excluded from
//     the grid import precisely because GenesisImportAgreement reconstructs them,
//     so admitting them would double-count - but the checkpoint was excluded with
//     them, and an unset grid attribute reads as zero.
//
//     Checkpoint() bills (currentBlock - checkpointBlock) * rate * aggregate
//     load, so the first checkpoint after a restore billed the entire height of
//     the chain against the full reconstructed load. SweepRevenue clamps to what
//     the pool holds rather than failing, so the outcome was not an error: it was
//     every consumer's collateral swept into the provider's earnings pool in one
//     transaction, and that collateral is the escrow backing service not yet
//     rendered. The provider-collateral-solvency invariant would have reported it
//     only after the money moved.
//
//     A restored checkpoint may sit below the genesis height and is left alone -
//     a restart at H+1 carrying a checkpoint of H owes exactly one block, and
//     flooring it would forgive service already rendered. Only a provider with no
//     checkpoint row at all is stamped, which says it is paid up to the moment
//     the chain starts - the only reading consistent with the agreements it is
//     starting with.
//
//   - At most AgreementExpirationBucketCap agreements may share one expiration
//     height.
//
//     AgreementExpirations tears down every agreement indexed at the current
//     height inside the EndBlocker, which has no gas meter, and each one
//     checkpoints its provider, moves money, destroys an allocation and rewrites
//     indexes. The consumer chooses the duration and therefore the height, so
//     agreements opened across many earlier blocks could be aimed at one block.
//
//     The bound is on creating that work rather than on doing it, which is the
//     opposite of how GridCascade is bounded in this same release, and the
//     difference is forced rather than stylistic. A cascade can stop half way and
//     resume next block. An expiry cannot: the index is read at exactly one
//     height, so an agreement missed there is never revisited, and until it is
//     its capacity stays in the provider's aggregate load where Checkpoint()
//     bills it against every other consumer's escrow. Deferring an expiry is not
//     slower settlement, it is somebody else paying for it - which is what the
//     agreement-expiry-liveness invariant exists to catch.
//
//     GAME RULE CHANGE: AgreementOpen, AgreementDurationIncrease and both
//     capacity changes can now be refused because the expiration height they
//     would land on is full. The remedy is to shift the duration by a block. An
//     agreement already indexed at a height keeps its slot, so a capacity change
//     that re-bases back onto the same block is never refused for this.
//
//     There is no counter and no migration. The check counts the bucket and
//     stops at the cap, so it costs at most cap+1 reads however large the bucket
//     is, and a chain arriving with heights already past the cap simply accepts
//     nothing new at those heights until they drain.
//
//   - A zero-height export rebases every agreement onto the new chain's clock,
//     and an import refuses an agreement that could never expire.
//
//     StartBlock and EndBlock are absolute heights on the agreement record, so an
//     export carries them through untouched. That is correct for a
//     height-preserving restart and a gift on a zero-height one: an agreement
//     exported at height H with E-H blocks left was re-indexed to expire at E on
//     a chain restarting near 1, supplying capacity for roughly the whole age of
//     the old chain that nobody posted collateral for.
//
//     prepForZeroHeightGenesis now checkpoints every provider against the old
//     clock - settling the span actually served while that clock still means
//     something - then rewrites each agreement to StartBlock 0 and EndBlock
//     equal to its remaining duration, and rebases the provider checkpoints to
//     match. The two clocks have to move together or Checkpoint() bills a span
//     the agreements no longer claim.
//
//     An agreement with nothing left gets one block rather than zero: an expiry
//     gets exactly one attempt, and a row indexed at a height the chain never
//     reaches would hold capacity in the provider's load forever.
//
//     InitGenesis panics on an agreement whose end block precedes the genesis
//     height, which is the same condition seen from the other side and catches a
//     zero-height export started at an initial_height the rebase did not expect.
//     Equal to the genesis height is fine; that agreement expires on the first
//     block.
//
//     Agreements and provider checkpoints are the only absolute heights that
//     survive a structs import. Planet and struct block clocks are exported but
//     never read back, grid lastAction is filtered out, the reactor charter stamp
//     is recomputed from params by RestampReactorCharterEligibility, and
//     CharterAge clamps an anchor ahead of the height to zero.
//
//   - A destroyed allocation is no longer returned as a live cache hit, and
//     destroying one twice is a no-op.
//
//     Destroy defers both removals to CommitAll, so the cache stays in
//     cc.allocations and the record stays in the store. GetAllocation's cache-hit
//     path returned that cache with found=true, and every caller reads the
//     boolean as existence - the same shape as the phantom PlayerCache, where a
//     context getter is a cache allocator and says nothing about whether the
//     object is real.
//
//     What made it damaging is that Destroy never zeroes the power attribute,
//     and GetPower reads the grid. A second Destroy therefore saw the original
//     power and took it off the source's load again - load that belongs to
//     whichever allocations are still sharing that source. Measured on a source
//     carrying 250 with 100 destroyed: the correct 150 became 50, so the source
//     advertised 100 of headroom it did not have, which is the direction that
//     oversubscribes a grid rather than the direction that wastes it. SetPower
//     on the same stale cache drove it to 40.
//
//     Fixed at the getter, which is what stops callers acquiring a stale handle,
//     and again at Destroy and SetPower, which is what covers a handle acquired
//     before the destroy. The auto-resize path already separated "allocation
//     missing" from "resize failed"; a destroyed allocation now correctly reads
//     as the first, so a stale hook sheds load instead of resizing rubble.
//
//     The specific chain reported against this - a stale auto-resize index
//     naming a destroyed allocation - was closed in v0.21.0 by keying that
//     index clear off the source object id. This is the primitive underneath it.
//
//   - A struct can no longer be assigned as its own defender, and any existing
//     self-registration is cleared.
//
//     StructDefenseSet never compared the two ids, and IsProtecting compares
//     locations - a struct is trivially co-located with itself - so nothing
//     refused the pairing. The damage is in the resolution order: StructAttack
//     runs defender counters, then the volley, then the target's own counter, so
//     a target counters only after surviving the shots. A self-registered target
//     was picked up in the defender pass instead - GetStruct returns one cache
//     instance per id, so the "defender" is literally the target - and its
//     counter landed before the volley. A counter that destroyed the attacker
//     then voided the volley outright and the target took nothing at all.
//     CounterSpent lives on the per-transaction cache, so it worked on every
//     attack.
//
//     GAME RULE: this restores the documented sequence rather than changing it.
//     structs.ai already describes defender counters, then the volley, then the
//     target's counter, and describes a defender as co-located with the struct
//     it protects. Worth clarifying upstream that self-assignment is refused
//     outright, which the doc only implies.
//
//     Refused at registration, and skipped again in ResolveDefenders, which is
//     what covers rows already on disk. The attacker is skipped there too, for a
//     structural reason rather than a game one: it would be countering itself.
//
//   - Genesis import refuses a fleet whose id cannot be parsed, and
//     GenesisState.Validate rejects the file first.
//
//     GetFleetById resolves a fleet by parsing its id into an index. A malformed
//     one returned a FleetCache that was never inserted into cc.fleets, and
//     CommitAll commits only what is in that map - so GenesisImportFleet
//     discarding the error dropped the fleet in total silence, while the players
//     and structs pointing at it imported normally. The result was an owner
//     holding fleet-bound assets with no fleet, and nothing anywhere said so:
//     the import returned no error to discard, and FleetList was not validated.
//
//     Fleets were the only import with this shape. Every other GenesisImport
//     takes a plain allocator that always caches and always commits, so a bad id
//     there is written under a bad key rather than lost.
//
//     The id rule now lives in types.ParseFleetId, used by both the keeper and
//     GenesisState.Validate. One function rather than two spellings, because the
//     property that matters is that a file which validates also imports.
//
//   - Grid attribute addition saturates at MaxUint64 instead of wrapping.
//
//     SetGridAttributeIncrement and the addition inside SetGridAttributeDelta
//     were the only arithmetic here without a guard; both siblings already
//     clamped their subtraction, which made these the odd ones out rather than a
//     considered choice. Capacity, load, stored ore and the replay nonces are
//     all treated as monotonic by their callers, so rolling over to a small
//     number destroys accounted value, understates committed load, and in the
//     nonce case resets replay protection.
//
//     Saturating rather than returning an error is deliberate: genesis import,
//     staking hooks and the block hooks have nowhere to put a rejection, and an
//     error every caller has to drop is a worse contract than a bound none can
//     exceed. The clamp logs, so a saturated attribute is diagnosable.
//
//     NOT reachable today, and the fix does not pretend otherwise. The only
//     compounding path is a cycle of allocations feeding each other, and that
//     was measured growing linearly rather than exponentially - roughly 10^17
//     rounds from a realistic seed. That cycle mints capacity and is a real bug,
//     tracked separately; this is only the backstop under it.
//
//   - The validator slashing hook queues its reactor instead of reconciling
//     every delegation inline.
//
//     BeforeValidatorSlashed ran ReactorUpdateInfusionsFromSlashing
//     synchronously over the validator's entire active delegation set - a
//     delegation read, an unbonding-delegation read and a grid rewrite each -
//     inside BeginBlock, which has no gas meter. The set is unbounded and its
//     size is chosen by the delegators. The SDK's own unbounded work during
//     Slash covers unbonding delegations and redelegations, a window-limited
//     set; this walked the full active one, so it was exposure the SDK does not
//     have rather than exposure it already had.
//
//     A slash leaves delegation shares untouched, so AfterDelegationModified
//     never fires for it and this hook is the module's only signal that one
//     happened. Reconciliation can be deferred but never dropped: it now goes
//     through ReactorSlashReconcileQueue, ReactorSlashReconcileBudget infusions
//     per block in the EndBlocker, resuming from a stored cursor.
//
//     Deferring also removed a piece of hand-arithmetic. The hook fires before
//     staking applies the slash, which is why the old code derived the
//     post-slash token figure itself; by the time the queue runs, live staking
//     state is the answer and nothing has to be carried.
//
//     GAME RULE CHANGE: an infusion not yet reached still carries its pre-slash
//     fuel and the grid capacity standing on it. The window is bounded by the
//     slash fraction and by delegators/budget blocks, and it is deliberate - the
//     alternative, zeroing the reactor's contribution up front, would brown out
//     every delegator's grid over a slash that took five percent.
//
//     The queue is not exported at genesis and does not need to be: genesis
//     import rebuilds every reactor's infusions from staking through
//     GenesisImportReactorInfusions, which is a full reconciliation by itself.
//
//   - The permission-on-object handlers resolve their target player, and a
//     permission record that goes to zero is removed rather than stored.
//
//     PermissionGrantOnObject, PermissionSetOnObject and PermissionRevokeOnObject
//     fed msg.PlayerId straight into GetObjectPermissionIDBytes, which is half of
//     a KV key. Nothing resolved it: the object check speaks to the object, and
//     PermissionCheck's owner shortcut passes an owner acting on their own object
//     whatever target they name. Any string at all was therefore storable.
//
//     Revoke was the sharpest of the three because it needs no real permission to
//     write anything: a bit the target never held is removed to zero, and zero
//     was committed as eight bytes rather than as an absence. Structs messages
//     are free, so that was forty permanent rows per player per block under
//     attacker-chosen keys, revoking nothing.
//
//     Both halves are fixed. The handlers now resolve through
//     cc.GetExistingPlayer, and SetPermissions treats Permissionless as a delete
//     while PermissionRemove writes nothing when the bits were already absent -
//     a missing key already reads as Permissionless, so a stored zero says
//     exactly what an absent row says and costs state forever.
//
//     This is the phantom-cache rule reached by a road that has no getter on it,
//     which is why TestArch_HandlersResolveMessagePlayerIdsThroughGetExistingPlayer
//     did not catch it: the test looked for a message id reaching cc.GetPlayer,
//     and these handlers never resolved the id at all. It now also fails a
//     handler that builds a permission key from a message-supplied player id
//     without resolving it. Note the detector only matched method calls, and
//     GetObjectPermissionIDBytes is a plain function - that shape blindness was
//     the whole of the gap.
//
//   - MsgStructTrash is registered in ChargeMessages.
//
//     Trashing a struct costs the same charge as building it, and the handler
//     both checks the owner's charge and calls Discharge(). The ante reads the
//     map rather than the handler, so with no entry it applied neither the
//     charge floor in StructsDecorator nor the per-transaction duplicate check
//     in ThrottleDecorator. CheckTx does not run handlers, so a transaction
//     carrying the same trash twice was admitted and then failed in delivery
//     once the first message had destroyed the struct - free to submit, a pure
//     Structs transaction paying no fee, and rolled back only after the block
//     had paid to execute it.
//
//     Nine handlers call Discharge(); the map listed eight. That is the shape a
//     missing maps.go entry always has - it fails silently rather than erroring -
//     so TestArch_DischargingHandlersAreChargeMessages now walks the handler
//     sources and requires the two sets to match in both directions. An entry
//     with no discharge behind it is a defect too: it spends the player's block
//     slot for nothing, and the two ante checks then disagree about whether the
//     slot was used.
//
//   - A substation can only be created from an allocation that is not already
//     feeding one.
//
//     SubstationCreate checked that the allocation existed and that the caller
//     could connect it, and nothing more. SetDestination is a move - it
//     decrements the old destination's capacity and hands the power to the new
//     one - so the same allocation could be submitted repeatedly, each call
//     minting a fresh substation and a fresh permission record and leaving the
//     previous substation behind with nothing feeding it. Creation is free, so
//     the only bound was the per-block message cap: forty permanent, unfunded
//     substations per player per block, from one allocation.
//
//     The invariant is not new. AllocationTransfer already refuses a connected
//     allocation, and the simulator only ever offered ones whose DestinationId
//     was empty - it was missing on the creation path alone. Enforced inside
//     NewSubstation rather than the handler so no future caller can skip it, and
//     ahead of the id counter so a rejection consumes nothing.
//
//     GAME RULE: an allocation released with SubstationAllocationDisconnect is
//     free to seed a new substation again. The rule is about being connected,
//     not about having been used.
//
//   - Permission cleanup after an object is destroyed is bounded per block, with
//     the remainder carried in PermissionCleanupQueue.
//
//     Destroying an object clears every permission granted on it, and the count
//     of those is chosen by whoever owns the object. Agreement expiry reaches
//     that cleanup from the EndBlocker, which has no gas meter, and the expiry
//     height is chosen by the consumer when they open the agreement - so the
//     cost of one block was whatever had been granted before it, materialised at
//     once as key copies, returned strings and retained events. The guild-rank
//     register clear had the same shape and emits up to PermissionBitCount
//     events per row.
//
//     Deferring is safe here in a way it is not for an agreement expiry, and the
//     difference is worth stating: a permission is only ever consulted after its
//     object has been loaded, and a destroyed object cannot be. Rows left behind
//     grant nothing and nothing accrues while they wait, so this is garbage
//     collection rather than settlement and needs no cursor - deletion is
//     destructive, so the next pass re-reads the prefix and finds what is left.
//
//     The cache is evicted by prefix rather than by the list of rows actually
//     deleted. A cache entry marked Changed is written back at CommitAll, so a
//     permission *created* in the same operation that destroys the object -
//     present in the cache, absent from the store, therefore absent from that
//     list - survived the clear and was written back afterwards. On an object
//     small enough not to be queued nothing ever came back for it.
//
//     Note this bound is what remains of the reported issue after the
//     permission-on-object handlers began resolving their target player: an
//     attacker can no longer mint a row per arbitrary string, only one per
//     registered player.
//
//   - Query pagination is bounded, and CountTotal is refused.
//
//     The query endpoints take no authorization and passed req.Pagination
//     straight to query.Paginate. Limit is a uint64 and is not capped, so one
//     request could ask a node to decode and retain an entire collection;
//     Offset+Limit is added without an overflow check; and a request with no
//     Limit switched CountTotal on, which walks the whole prefix after the page
//     has been collected.
//
//     Limit is now capped at QueryPageLimitMaximum and CountTotal is forced off
//     at all thirty query.Paginate sites, through one helper, with
//     TestArch_PaginatedQueriesAreBounded holding new queries to it.
//
//     CountTotal is the half that actually bounds the work - it scans the prefix
//     however small the page - so capping the page alone would not have been
//     enough. Refusing it means pagination responses report total as 0.
//
//     PUBLIC API CHANGE: total is now always 0. This is not a new contract so
//     much as the universal case of an existing one - the SDK already ignores
//     count_total whenever a key is set, so every client paginating by next_key
//     saw 0 already - but a client reading it will see a change. The set stays
//     fully reachable by following next_key. GRASS and structs-pg should be
//     checked for any use of pagination.total.
//
//     This bounds the node serving the query rather than consensus. No chain
//     state is involved; what it protects is a node exposing its query
//     endpoints against whatever its largest collection happens to be. Setting a
//     non-zero QueryGasLimit in app.toml is worthwhile alongside this, since it
//     also covers the SDK's own modules, which have the identical shape.
//
//   - Genesis import restores the pending struct destruction queue, rescheduled
//     onto the restarted chain's clock.
//
//     Destroying a struct does not remove it: the flag is set, the struct stays
//     in its planet or fleet slot, and StructSweepDestroyed clears the slot and
//     deletes the object StructSweepDelay blocks later. That appointment was
//     exported and never imported - SetStructDestructionQueueAtHeight existed
//     with no caller - so a struct destroyed within those five blocks of an
//     export came back as rubble nothing was scheduled to collect, holding its
//     slot and, for a command ship, Fleet.CommandStruct, permanently. There was
//     no way out by hand either: StructTrash refuses an already-destroyed
//     struct.
//
//     Rescheduled rather than restored verbatim, because StructSweepDestroyed
//     reads the queue at *exactly* the current height - the same one-attempt
//     shape as agreement expiry. An exported appointment is either behind the
//     restart height already or, after a zero-height export, so far ahead the
//     chain never reaches it; either way the entry would never be visited. The
//     struct is rubble and contributes nothing, so when it is collected does not
//     matter, only that it is.
//
//   - Genesis import restores the BuildDraw reservation held by a pending build.
//
//     InitiateStruct charges BuildDraw against the owner's load the moment a
//     build starts, and only StructBuildComplete or DestroyAndCommit gives it
//     back. The import rebuilds a player's load from nothing -
//     GenesisImportPlayer resets it to PlayerPassiveDraw, and structsLoad is
//     deliberately excluded from the grid import because it is derived - and
//     GenesisImportStruct added load only through GoOnline, which a build in
//     flight never reaches. The reservation simply was not there afterwards.
//
//     Under-counting is only half of it. StructBuildComplete decrements
//     BuildDraw unconditionally when the build finishes, and
//     SetGridAttributeDecrement clamps at zero, so completing an imported build
//     would take that load out of the player's passive draw and their other
//     structs instead - leaving them running more than their power supports.
//
//     Scoped to unbuilt structs. A built one has already released the
//     reservation, and a destroyed one returns from the branch above before
//     reaching this, which is right: destruction released it too.
//
//   - An unfinished build restarts its proof clock at the genesis height rather
//     than at block zero.
//
//     BlockStartBuild is the age a build proof is priced from:
//     StructBuildComplete computes currentHeight - BlockStartBuild and
//     CalculateDifficulty falls with that age, clamping at 1. Importing zero did
//     not mean "no progress", it meant "as old as the chain" - so on a
//     height-preserving restart of a mature chain, every build in flight arrived
//     with its puzzle already collapsed to the minimum. Nothing downstream
//     refuses a zero start; the raid path carries such a guard and this one does
//     not.
//
//     The genesis height is the only value safe in both directions. Carrying the
//     exported start would be more faithful on a height-preserving restart and
//     unusable after a zero-height one, where it sits ahead of the new height and
//     StructBuildComplete refuses the proof permanently. Resetting costs the
//     builder their accumulated age, which asks for more work rather than less.
//
//     GAME RULE CHANGE: a build in flight across a genesis import loses its
//     accumulated age and starts its proof clock again.
//
//     This is the same if/else as the BuildDraw fix above, and the online arm
//     already stamped the genesis height. Only the offline one wrote zero, so the
//     two have collapsed into one.
//
//   - A fleet id has exactly one spelling. ParseFleetId refuses anything but the
//     canonical form.
//
//     A fleet is the only object identified by a parsed number rather than by
//     its id text: GetFleetById parses the suffix and GetFleet keys the cache and
//     the store by that index. Structs, players and substations are keyed by the
//     raw string, so a re-spelt id simply fails to load - fleets were the
//     exception, and ParseUint accepts leading zeros, so "9-1", "9-01" and
//     "9-001" all reached one fleet while remaining three distinct strings.
//
//     That mattered wherever the string rather than the fleet is the identity.
//     The per-fleet throttle keys off the wire value, so three spellings bought
//     three moves of one fleet in a block where the rule is one, and the raid
//     proof throttle keys off the same field.
//
//     Fixed at the parser rather than at the throttle key. Canonicalising the key
//     would have left the aliases resolving and merely counted them together;
//     refusing them means there is one spelling of a fleet everywhere - in a
//     throttle key, in an event, in a client's records - rather than one
//     canonical form and an unknown number of accepted aliases. The throttle
//     extractor therefore still keys off the raw string, and its comment records
//     that relaxing the parser reopens this.
//
//     GAME RULE: a fleet id must be exactly "9-<decimal, no padding>". Nothing
//     the chain generates is anything else, so no client sending back what it was
//     given is affected.
//
//   - Ante rejections during checkTx and reCheckTx are logged at DEBUG rather
//     than ERROR. A deliverTx rejection stays at ERROR.
//
//     observeReject wrote one ERROR line carrying the error text for every
//     rejection in every phase. A rejected transaction advances no sequence and
//     consumes no throttle count, so the same correctly signed over-cap
//     transaction can be replayed indefinitely - one line each, at no fee - and
//     CometBFT's check_tx RPC bypasses the mempool cache that would otherwise
//     deduplicate it. The volume of an operator's ERROR log was therefore set by
//     whoever was sending to the node.
//
//     A deliverTx rejection is different in kind: it cannot be produced faster
//     than blocks are, and it means a transaction got past admission and failed
//     anyway, which is the case worth reading. That is where the line stays.
//
//     No signal is lost. The telemetry counter fires in every phase and already
//     carries a phase label, so "reject rate by code" and anything alerting on
//     it - including the incident-2026-05 watch on code 2020 - behaves exactly
//     as before. What stops is one disk line per attempt.
//
//     This is node-local: no consensus state is involved, and logging is not
//     part of the state machine.
//
//   - The per-player message cap is now applied at admission as well as at
//     delivery, against a separate node-local count.
//
//     The authoritative counter lives in the transient store and is touched only
//     in DeliverTx, because counting a transaction at CheckTx and again when it
//     is delivered would charge the quota twice. That left admission bounded per
//     *address* by CheckTxThrottleDecorator while the cap it is predicting is per
//     *player*, and address associations only converge on a player inside
//     StructsDecorator. A player with several registered addresses could
//     therefore fill the mempool with transactions the delivery cap was always
//     going to refuse - free to send, since Structs transactions pay nothing,
//     and paid for in block bytes and validator time.
//
//     The admission count is a plain in-memory map reset when the height moves,
//     the same shape CheckTxThrottleDecorator already uses for addresses. It
//     never touches the transient store, so it cannot double-charge the
//     authoritative counter or make one node's block differ from another's;
//     nodes may admit slightly differently, which is already true of the address
//     throttle beside it.
//
//     ReCheckTx and simulate are excluded for the same reasons that decorator
//     gives: both run on transactions that already passed fresh CheckTx, so
//     counting them again would evict transactions legitimately admitted.
//
//   - A zero-height export with a jail allowlist no longer kills the exporter.
//
//     Excluding a validator set its Jailed field and stored the record. The
//     staking power index is a separate structure with its own setter and
//     deleter, so that left a jailed validator in the power store, and
//     ApplyAndReturnValidatorSetUpdates walks it and refuses - "should never
//     retrieve a jailed validator from the power store". The error went to
//     log.Fatal, which exits without unwinding, so excluding any active
//     validator inside MaxValidators produced no genesis at all. That is the
//     worst moment for it: a planned migration, the chain already stopped, and
//     nothing to show for the export but an exit code.
//
//     prepForZeroHeightGenesis now removes the power-index entry first, checks
//     the errors it was discarding, and returns them rather than calling
//     log.Fatal - including the one this release added for the agreement rebase.
//     ExportAppStateAndValidators already returns an error, so it just
//     propagates.
//
//     The SDK-boilerplate panics further down the same function are left as
//     they are: a panic in a CLI export is loud and leaves a stack trace, which
//     log.Fatal does not.
//
//     Tooling only. Nothing here runs on a live chain.
//
// This upgrade carries three state migrations.
//
// MigrateGridCascadeQueue re-keys the pending cascade queue by sequence. It runs
// first, before anything else in the upgrade can enqueue: the re-key reads every
// row under the queue prefix as a legacy object-id key, and object ids are not
// fixed width, so an id eight bytes long becomes indistinguishable from a
// sequence the moment the two shapes coexist. Legacy rows carry no ordering of
// their own and are re-appended sorted, which is the order the old cascade
// processed them in.
//
// MigrateSelfDefenseRegistrations clears any struct registered as its own
// defender. The runtime guard already makes such a row inert, so this is not
// what closes the exploit; it matters because the row is also a registration
// slot, and a struct defends one target at a time, so leaving it would keep that
// struct's real assignment blocked.
//
// MigrateInfusionFuelRounding streams each reactor's infusions rather than
// collecting them. An upgrade's work is unavoidable - it runs at a coordinated
// height with an infinite gas meter against live cardinality - but holding a
// reactor's whole delegator set in decoded records at once is not, and running
// out of memory in an upgrade block is a worse failure than a slow one, since
// the block is retried from the same state. Only the addresses are held, because
// the reconcile writes and the iterator has to be closed before it runs.
//
// MigrateInfusionFuelRounding recomputes every reactor infusion against the
// corrected conversion. Without it the fix would only reach an infusion the next
// time staking touched that delegation, and a delegation nobody moves is never
// touched. Capacity moves down where it moves at all, and the grid cascade that
// follows sheds the allocations that were only ever powered by rounding.
//
// Everything else in this upgrade is binary-only, and the cascade queue re-key
// is the only store-key change. The address nonce store starts empty and every
// address correctly begins at 0, because the sign-byte change invalidates every
// previously issued proof regardless. The upgrade exists so validators adopt the
// new ante and handler behaviour at a coordinated height - nodes running a mix
// of the old and new binaries would accept different transactions and diverge.
const UpgradeName = "v0.22.0"
