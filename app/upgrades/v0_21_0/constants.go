package v0_21_0

// UpgradeName is the on-chain upgrade plan name for the v0.21.0 binary.
//
// v0.21.0 expands the guild bank with alpha<->token conversion and per-guild
// convert fees. It carries both consensus/behavior changes and one-time state
// migrations.
//
// Consensus / handler changes (binary):
//
//   - New MsgGuildBankConvert converts ualpha into a guild's alpha-backed token
//     at the current collateral ratio. It is ratio-preserving: the full deposit
//     (including the convert-in fee) is added to collateral and tokens are
//     minted against the net alpha, so the alpha-backing-per-token of existing
//     holders never decreases. Requires supply > 0 AND collateral > 0 (a zero
//     on either side leaves the ratio undefined and, for zero collateral, would
//     allow unbounded minting); bootstrapping a fresh bank stays on the admin
//     MsgGuildBankMint path. A nonzero minAmountToken is required to guard
//     against ratio and fee movement earlier in the block.
//
//   - New MsgGuildBankConvertToken converts one guild token into another in a
//     single atomic transaction (source token -> ualpha -> target token),
//     applying the source guild's convert-out fee and the target guild's
//     convert-in fee. It is the composition of a redeem and a convert leg; each
//     leg emits its own event. A same-guild convert is rejected.
//     minAmountToken is required and guards the final target-token output.
//
//   - New MsgGuildUpdateBankConvertInFee / MsgGuildUpdateBankConvertOutFee set
//     the two per-guild fee rates (LegacyDec, range 0.0 inclusive to 1.0
//     exclusive). The fee is always
//     retained in the guild's collateral pool, raising backing-per-token for
//     all holders. Gated by PermAdmin on the guild object (reusing the existing
//     admin bit; no new permission).
//
//   - MsgGuildBankRedeem now applies the guild's convert-out fee and requires a
//     nonzero minAmountAlpha slippage guard. Its pro-rata payout is also
//     recomputed in pure integer math (floor(amountToken * collateral / supply))
//     instead of the previous divide-first LegacyDec formula. The prior formula
//     used banker's rounding after dividing, which could overpay the redeemer by
//     up to ~0.5e-18 * collateral (breaking the pool-favored rounding invariant
//     once collateral approached ~2e18). The integer form always rounds in the
//     pool's favor. This is a behavioral consensus change to an existing message.
//
//   - Guild-bank inputs, collateral, supply, bridge values, and outputs must fit
//     uint64, matching the transaction and event schema. Values outside that
//     range are rejected before arithmetic or state mutation.
//
//   - StructType gains canDefend (fleet=true, planetary=false). StructDefenseSet
//     and combat defender resolution reject non-defending types.
//
//   - Ore mine/refine clocks move from StructAttributes onto PlanetAttributes.
//     Rigs on a planet share one mine clock and one refine clock.
//     oreMiningActiveQuantity / oreRefiningActiveQuantity are the authoritative
//     on/off signals. Mining and refining are blocked while LocationListStart
//     is set (planet under raid); on raid end the clocks are shifted forward by
//     the paused duration so difficulty age is preserved.
//
//   - Reactor energy is gated on validator health. A reactor whose validator is
//     jailed (or has vanished from staking) runs at a zero fuel-to-energy ratio,
//     which zeroes both the operator's commission energy and every delegator's
//     share. Fuel and the underlying delegations are untouched, so no stake
//     moves. The gate is applied by AfterValidatorBeginUnbonding during staking's
//     EndBlock, which is the same block the jail happens in, and the resulting
//     capacity drop cascades through the grid in that block. Recovery runs
//     through AfterValidatorBonded on rebond, or through the new
//     MsgReactorRestart for an operator who unjails but stays below the
//     active-set cutoff. Restoration reconciles against live staking state, so
//     a validator slashed while jailed returns proportionally less energy.
//
//   - New permissionless MsgReactorRestart reconciles a reactor's infusions with
//     live staking state. It writes only derived state, so it carries no
//     ownership or permission requirement beyond the standard ante-level player
//     registration, and it force-gates a still-jailed reactor rather than
//     reviving one.
//
//   - Infusion emptiness now requires Fuel == 0 in addition to Power == 0 and
//     Defusing == 0. Without this, a ratio-zero infusion would be treated as
//     empty and destroyed by the EndBlock destruction queue, stranding the
//     delegator's fuel.
//
//   - Removing a delegation outright now clears the infusion behind it.
//     BeforeDelegationRemoved was a no-op, and staking's Unbond routes a
//     delegation whose shares reach zero through RemoveDelegation, which fires
//     only that hook and skips AfterDelegationModified. A full redelegation
//     therefore left the source infusion's Fuel, Power and grid capacity in
//     place while the destination was granted capacity for the same stake, so
//     one stake produced energy at two reactors. GuildMembershipJoin redelegates
//     a player's entire infusion, so this accrued through ordinary play, not
//     just deliberate abuse. Defusing is left alone, since an unbonding balance
//     outlives the delegation record. The fuel is zeroed explicitly rather than
//     by reconciling, because the hook fires before staking deletes the row and
//     a reconcile would rewrite the stale value it was called to clear.
//
//   - PlayerUpdatePrimaryAddress now requires the caller to hold PermAll. The
//     handler grants PermAll to the incoming address and moves the player's
//     balance and delegations with it, so a narrower PermAdmin gate was a
//     privilege-escalation path. The ante PermissionMap entry matches.
//
//   - MsgAgreementOpen now requires PermTokenTransfer on the signing key, under
//     every provider access policy. The collateral is debited from the player's
//     primary address, making it a token spend, but the open-market branch checked
//     only that the signer mapped to a registered player and the guild-market
//     branch checked PermProviderOpen, which grants access rather than spending
//     authority. A registered secondary key with neither bit could therefore
//     commit the primary balance. The message moves out of
//     DynamicPermissionMessages into PermissionMap so the ante enforces the bit as
//     well; the access-policy branch stays in the handler.
//
//   - MsgAgreementDurationIncrease likewise now requires PermTokenTransfer in
//     addition to PermUpdate. Extending an agreement buys the extra blocks out of
//     the primary address, and update rights on the agreement said nothing about
//     whose money may move, so a secondary key with PermUpdate could top up
//     duration from the primary balance. The two agreement handlers were the only
//     signer-authorized debits of the primary missing a token bit;
//     TestArch_PrimaryAddressDebitsRequireTokenBit in app/ante is now the standing
//     guard for the rest of the family.
//
//   - An agreement now starts serving in the block it is opened rather than the
//     block after. AgreementOpen raises the provider's load immediately and
//     checkpoints the provider at the opening height, and Checkpoint bills
//     aggregate load from the checkpoint block, so starting a block later charged
//     the provider for one block of service the consumer never received and left
//     the collateral pool short by that much. endBlock is therefore one block
//     lower than the old binary would have written for the same duration; the
//     duration itself, the collateral charged, and the unearned collateral at the
//     opening block are all unchanged.
//
//   - An agreement expiry no longer abandons its teardown when the allocation it
//     meant to destroy has already left the store. Expiry runs from the EndBlocker,
//     which reads the expiration index at exactly the current height, so there is
//     no range scan and no retry: an agreement not torn down on the one block it
//     comes up was never revisited, kept its capacity in the provider's load, and
//     had Checkpoint bill that capacity against the shared collateral pool every
//     block afterwards, out of other consumers' escrow. A missing allocation is
//     nothing to tear down, which destroyAllocation already assumed on the path
//     where CurrentContext has never heard of it; the case where it is cached from
//     earlier in the operation and only Destroy's re-read notices now behaves the
//     same way. This is the last reachable failure inside Expire, so an expiry can
//     no longer strand its own agreement.
//
//   - A forced close driven by the allocation going away — a grid brownout, a
//     destroyed struct, a deleted allocation — now checkpoints the provider before
//     it settles, matching the consumer, provider and expiry paths. Checkpoint
//     bills the current agreement load across the span since the last checkpoint,
//     so settling first meant the revenue earned over that agreement's own
//     lifetime was never swept: it stayed in the collateral pool once the load was
//     decremented and the agreement removed, unreachable by any later checkpoint
//     or withdrawal, both of which are computed from load. The provider now
//     receives that revenue and the pool empties to exactly what it owes.
//     provider-collateral-solvency cannot see this case, since stranded revenue
//     leaves the pool over-funded rather than short; the guards are
//     TestTeardown_AllocationDrivenSweepsEarnedRevenue and
//     TestArch_TeardownPathsCheckpointBeforeMutating, which holds every path that
//     calls beginTeardown to the same ordering.
//
//   - A substation carrying more load than capacity now reports zero available
//     capacity instead of a wrapped one. Load above capacity is a normal,
//     reachable state — an allocation feeding a substation can shrink or vanish
//     mid-block and only GridCascade in the EndBlocker forces it back, later
//     still if it cannot shed enough — but SubstationCache.GetAvailableCapacity
//     computed capacity - load in uint64, so throughout that window it answered
//     with nearly 2^64 rather than nothing. AgreementCapacityIncrease is the
//     reachable consequence: its only headroom gate was that value, and
//     AllocationCache.SetPower, unlike SetInitialPower and SetDynamicPower, does
//     no check of its own, so the consumer who opened an agreement could enlarge
//     it up to the provider's published capacity maximum on a substation with
//     nothing left to give, deepening a brownout that GridCascade then had to
//     resolve by destroying other players' allocations. Such an increase is now
//     rejected with the existing exceeds_available error. AgreementOpen read the
//     same wrapped value through ProviderCache.AgreementVerify but was never
//     exploitable, because creating the allocation goes through SetInitialPower,
//     which refuses on its own correct comparison; it now fails earlier and with
//     that error instead. PlayerCache.GetAllocatableCapacity got the same clamp
//     for consistency with its already-guarded siblings. No migration: an
//     over-subscribed grid is self-correcting, since GridCascade sheds load by
//     destroying allocations, so this leaves no corrupted stored aggregate.
//
//   - Destroying an automated allocation now actually clears its auto-resize
//     hook. The index is keyed by source object id, but AllocationCache.Destroy
//     cleared it with the allocation's own id, so the delete matched a key that
//     was never written and every automated allocation ever torn down left a hook
//     naming an allocation that no longer exists. Such a hook is not inert:
//     SetSource refuses any new automated allocation on a source the index
//     mentions, without checking that the allocation is still there, so the
//     source was bricked permanently; and the infusion capacity path took the
//     hook as proof that something was tracking the source, called AutoResize on
//     a missing allocation, and skipped the grid cascade that a capacity cut
//     should have triggered, leaving load unshed. Automated allocations reach
//     this through GridCascade, substation deletion, and power-generating struct
//     offline or destruction; MsgAllocationDelete only accepts dynamic ones.
//     AutoResizeAllocation now also distinguishes a missing allocation from a
//     failed resize, and the capacity path clears a hook of the first kind and
//     falls through to the cascade — a resize failure is not treated as a stale
//     hook, since that would shed load over a transient error.
//
//   - Deleting a provider now empties its collateral and earnings pools into the
//     owner's primary address, after every agreement has been closed and every
//     consumer made whole. Both pool addresses are derived from the provider id
//     and nothing but the provider record can reach them, so anything left at that
//     moment was unreachable for good — and there is normally something, since
//     every checkpoint and penalty payout truncates its share down and leaves the
//     remainder behind.
//
//   - A second invariant, agreement-expiry-liveness, breaks on any agreement still
//     in state after its end block. provider-collateral-solvency cannot see that
//     state: it clamps every span to the agreement's own window, so an overdue
//     agreement reads as one that has simply been fully served, and it only breaks
//     once the pool is already short.
//
//   - Agreement capacity changes re-verify the provider's advertised capacity and
//     duration ranges, which previously bound only at open. A change is rejected
//     if the resulting capacity falls outside capacityMinimum/capacityMaximum, or
//     if the rescaled remaining duration falls outside
//     durationMinimum/durationMaximum. Since a capacity change re-prices the
//     unearned span, this closes a decrease toward capacity 1 multiplying the
//     remaining duration by the old capacity. A change of zero is now rejected
//     instead of re-basing the accounting window for free, and both methods now
//     validate fully before releasing the voided cancellation penalty.
//
//   - A destroyed struct is now rejected by every operation for the rest of its
//     sweep window. Destruction flags the struct and leaves it built and
//     slot-resident until StructSweepDestroyed removes it StructSweepDelay blocks
//     later, and in that window StructActivate, StructBuildComplete,
//     StructBuildCancel, StructMove, StructGeneratorInfuse, StructDefenseSet and
//     StructDefenseClear all used to succeed. Activation was the damaging one:
//     GoOnline re-added the owner's load and the planet's shield and defensive
//     counters, and the sweep then deleted the struct without taking it offline,
//     leaving those behind permanently. All of them now fail with StructStateError
//     (1250) naming state "destroyed". DestroyAndCommit is idempotent, so a second
//     destruction of the same struct no longer releases its BuildDraw reservation
//     and type count a second time. StructSweepDestroyed takes a still-online
//     struct offline before removing it and logs at error level; that path is
//     unreachable through the guards above and the log is the signal that a new one
//     exists.
//
//   - Planet gains locationListExtra and locationListCount. Raid-queue capacity
//     is 1 + locationListExtra (protobuf default extra=0 => length 1).
//     MsgFleetMove onto a foreign planet whose queue is already at capacity is
//     rejected with FleetStateError queue_full. SetLocationToPlanet maintains
//     locationListCount on enqueue/dequeue.
//
//   - Name and pfp validation no longer reads the compiling toolchain's Unicode
//     tables. Go resolves \p{L} in a regexp, and unicode.Is against unicode.L,
//     Mn, Me or Cf, from the standard library of whichever toolchain built the
//     binary, and those tables grow with Go releases: U+088F is unassigned in
//     Unicode 15.0.0 (shipped by Go 1.23 and 1.24) and a letter in later
//     versions. Since nothing pinned a toolchain — go.mod's directive is a floor,
//     not a ceiling, and GOTOOLCHAIN=local opts out entirely — a name built from
//     such a code point was accepted by some validators and rejected by others,
//     and only the accepting side wrote it. Classification now runs against
//     checked-in Unicode 15.0.0 tables (x/structs/types/unicode_tables.go), so
//     the accepted character set is fixed by state rather than by build
//     environment. ValidatePlayerName, ValidateEntityName and ValidatePlanetName
//     keep their exact current character sets and error messages; code points
//     assigned after Unicode 15.0.0 stay rejected on every binary. Moving the
//     pinned version is itself consensus-breaking and needs its own upgrade.
//
//   - The same applies to ValidatePfp, in the opposite direction and not covered
//     by the report that prompted this: a URL path has no character allow-list,
//     so the format-category test decided acceptance for arbitrary runes. A code
//     point newly assigned to Cf was accepted by old binaries and rejected by
//     new ones. It is now pinned too. The pfp URI scheme is additionally compared
//     with ASCII-only folding instead of strings.ToLower and strings.EqualFold,
//     which read the toolchain's case tables and would, for instance, have
//     folded U+017F to "s". Every allowed scheme is ASCII, so the only names
//     this newly rejects are schemes containing a non-ASCII rune that case-folds
//     into ASCII.
//
//   - NormalizeName is pinned for a stronger reason than the validators: its
//     output is a KV key, not just a comparison value. SetGuildNameIndex stores
//     "Guild/name/" + NormalizeName(name), and RemoveGuildNameIndex deletes by
//     re-normalizing a name already in state, so a case-folding difference
//     between two binaries would write and delete different keys. Case folding
//     and space trimming now come from the pinned tables. norm.NFC stays, being
//     pinned by go.sum rather than by the toolchain; bumping golang.org/x/text
//     is therefore also consensus-breaking, and TestNFCGoldenVectors is the
//     guard. Output is byte-identical to v0.20.x for every input on any Go
//     1.23/1.24 build, proven exhaustively over the whole rune space by
//     TestPinnedTablesMatchToolchain and TestNormalizeNameMatchesLegacyForm.
//
//   - CurrentContext.CommitAll now commits every cache map in sorted key order
//     through commitCaches, instead of ranging the maps directly. Go randomizes
//     map iteration order per process, and AddressCache.Commit allocates an auth
//     account number from the account keeper's global monotonic sequence for any
//     address that does not have one, so two addresses committed in map order
//     were given different account numbers on different nodes. That is divergent
//     auth state and a different app hash, reachable from a genesis AddressList
//     with two unbacked addresses and from a single transaction carrying two
//     AddressRegister messages. Account allocation was the only order-dependent
//     write of the eighteen; every other Commit writes keys derived from its own
//     map key. Note that this changes two observable things at the release, not
//     just one: the account numbers assigned to addresses from this point on,
//     and the order of events emitted within a block, which GRASS and structs-pg
//     consume. No event field changes, and the previous order was random rather
//     than meaningful, so nothing could have depended on it.
//
//     There is no migration and there must not be one. Account numbers already
//     in auth state are correct for the order they were assigned in and must
//     never be renumbered, and a chain that actually hit the divergence would
//     already have halted rather than reached this upgrade. Like the rest of the
//     consensus changes above, this one is NOT gated by the upgrade handler: it
//     takes effect the moment the binary runs, so the release must be a
//     coordinated swap rather than a rolling one.
//
//   - SetPlayerIndexForAddress no longer provisions an auth account for an
//     address that fails bech32 parsing, and returns an error saying so. The old
//     form discarded the parse error, which left the empty AccAddress, found no
//     account for it, and so created and numbered an auth account for the empty
//     address — burning a sequence number on a non-address. The index row itself
//     is an ordinary KV write keyed by the address string, order-independent and
//     meaningful for any key, so it is still written; only the provisioning is
//     refused. GenesisState.Validate now rejects a malformed AddressList, which
//     is the only way such an address can arrive: AddressRegister and
//     GuildMembershipJoinProxy both require the address to equal PubKeyToBech32
//     of a proof pubkey, the staking hooks pass an AccAddress that was already
//     parsed, and everything else takes the SDK-validated msg.Creator.
//
//   - The ante no longer reserves an object-global throttle key for a signer
//     with no standing on the object it names. ThrottleDecorator built the proof
//     and operational keys straight from transaction fields —
//     proof/<structId>, proof/<fleetId>, fleet/<fleetId>, explore/<playerId>,
//     register/<playerId> — and wrote them before the handler ran. Nothing
//     upstream could catch that: StructsDecorator checks only the signing
//     address's own bits and a primary address holds PermAll, so naming somebody
//     else's struct passed. The reservation then survived the handler's
//     rejection, because the SDK commits the ante cache before executing
//     messages and only discards the message cache on failure. One free
//     transaction of MaxMsgCount messages could therefore park up to forty
//     victim objects per block, denying their owners fleet movement, planet
//     exploration, address registration and every proof-of-work completion,
//     every block, for nothing. A key is now written only once
//     Keeper.ThrottleTargetAuthorized agrees the signer could legitimately
//     consume it, resolving the named struct, fleet or player to its owner and
//     running the handlers' own PermissionCheck against it.
//
//     The reservation stays in the ante rather than moving to a post-handler,
//     which would look like the tidier fix and is not one: post-handler writes
//     go to the message cache and are discarded whenever a message fails, and a
//     proof slot that a failed attempt does not consume lets a player grind
//     nonces on-chain across many transactions in one block instead of mining
//     off-chain. A refusal skips the reservation and lets the message through to
//     its handler for the real error; it is deliberately not an ante reject,
//     since the ante sees pre-transaction state while a handler sees the state
//     earlier messages in the same transaction left, and a transaction that
//     grants a permission and then uses it must not be refused at admission. No
//     migration: throttle keys live only in the transient store, which is empty
//     at the upgrade height. The per-player charge/<playerId> key is unchanged,
//     being derived from the signer's own player index rather than anything the
//     transaction names.
//
//   - Guild join bypass levels outside the declared enum are now rejected on the
//     way in and denied on the way out. guildJoinBypassLevel declares closed,
//     permissioned and member, but proto3 enums are open — the generated decoder
//     shifts bytes into an int32 without consulting the enum — and the update
//     handlers wrote msg.GuildJoinBypassLevel straight through, so a value like
//     500 both stored and round-tripped. The three readers in guild_cache.go
//     then switched on it with no default, leaving err nil: CanRequestMembership
//     admitted the request, CanApproveMembershipRequest demanded nothing at all
//     where permissioned demands PermGuildMembership and member demands existing
//     membership, and CanInviteMembers did the same for invites. A player
//     holding only PermGuildJoinConstraintsUpdate could therefore open a guild
//     to any registered outsider, who submitted and approved their own request
//     and was migrated in at the entry rank — PermGuildMembership authority
//     obtained from a bit that is not PermGuildMembership.
//
//     Both halves changed. The readers end in a default that denies, which is
//     what covers records already on disk. The writers refuse an undeclared
//     level: GuildCache.SetJoinInfusionMinimumBypassByRequest / ByInvite now
//     return ErrInvalidGuildJoinBypassLevel (1508, registered since v0.20 and
//     until now unused) and both update handlers propagate it, checking ahead of
//     their equal-to-current guard so a corrupted record cannot be rewritten to
//     itself. GenesisState.Validate carries the same check, genesis being the
//     one path that assigns a Guild record without passing the setters.
//     Undeclared levels are refused rather than clamped at the handler, so a
//     client sending one learns it was wrong instead of silently getting closed.
//
//   - Approving, denying or revoking a guild membership application now requires
//     the application to exist. Membership is two-sided — a player files a
//     request and the guild approves it, or the guild issues an invite and the
//     player accepts it — and the stored application is the only evidence the
//     first leg happened. GetGuildMembershipApplicationCache synthesized a
//     proposed application on a store miss, which handed every approval path
//     precisely the consent it was about to verify. On
//     GuildMembershipRequestApprove that was a force-join: CanRequestMembership
//     takes no player argument and only asks whether the guild has requests
//     open, so any member of a recruiting guild named any victim and
//     ApproveRequest overwrote their GuildId, reset their GuildRank to the entry
//     rank and moved their substation connection, evicting them from the guild
//     they were in. The victim never transacted. Deny and revoke were milder —
//     the status they set becomes a delete at commit, so the write was a no-op —
//     but each reported success for an application that did not exist.
//
//     The loader is now two. GetOrCreateGuildMembershipApplicationCache keeps the
//     synthesis and belongs to the three handlers that open an application
//     (request, invite, join); GetPendingGuildMembershipApplicationCache requires
//     a stored row and belongs to the six that consume one, returning
//     ErrGuildMembershipApplication (1502, registered and until now unused) on a
//     miss. GuildMembershipApplicationCache.requirePending is the backstop on the
//     six terminal mutators, additionally requiring status proposed; DirectJoin
//     and Kick are deliberately outside it, each creating and consuming its own
//     record and carrying its own authorization. All six handlers also now check
//     the mutator's returned error, which they previously assigned and dropped —
//     harmless while every mutator returned nil, and the reason requirePending
//     needed the handlers repaired to take effect at all.
//
//     Same change closes the mirror on the creation side. GuildMembershipInvite
//     has no Verify call of its own, so the loader's CanInviteMembers is its
//     entire authorization, and that check ran only when synthesizing. An invite
//     already on file was therefore unguarded: an outsider named it and carried
//     on into SetSubstationIdOverride, which validates rights on the destination
//     substation and says nothing about the guild, pointing the invite at a
//     substation they own — and the invitee connected there on accept. The check
//     now runs on both branches, so amending an invite needs the same authority
//     as sending one.
//
//     Two residuals are accepted rather than fixed, recorded here so they are not
//     re-reported as the bypass above. First, that authority is equal to the
//     authority to create and not stricter, so at bypass level member — where
//     every member may invite — every member may also retarget a colleague's
//     pending invite. A guild wanting fewer amenders sets
//     JoinInfusionMinimumBypassByInvite to permissioned, which demands
//     PermGuildMembership on the guild object;
//     TestGuildMembershipInviteAmendmentFollowsInviteAuthority pins both halves.
//     Second, an invitee cannot bind their approval to the destination they were
//     shown, so an authorized amender can still change it under someone about to
//     accept. MsgGuildMembershipInviteApprove already carries a substationId, but
//     it is a setter routed through SetSubstationIdOverride and gated by
//     CanManageConnectionsBy, so naming the guild's own entry substation needs
//     connection rights there that a prospective member will not hold. Closing it
//     needs a separate read-only expectation field, which is a proto change and
//     the full ante-map walk. Bounded meanwhile by GetSubstationId falling back to
//     the guild's entry substation whenever the stored field is empty, so the
//     default path is always the guild's own.
//
//     GenesisState.Validate additionally rejects an undeclared guildJoinType or
//     registrationStatus on a stored application, genesis being the one path that
//     assigns the record without passing a setter. Both are authorization input:
//     the join type selects which side's consent the application stands for, the
//     status whether it is still live.
//
//     No migration, and the reason matters. A forced membership is byte-identical
//     to a consented one — MigrateGuild writes the same GuildId and rank either
//     way — so nothing on disk identifies a victim, and the exploit corrupts no
//     aggregate: guild membership is a field on the player, the substation move
//     goes through the ordinary connect path, and the synthesized application was
//     deleted at commit. There is nothing to recompute and no safe guess at what
//     to undo. Affected players rejoin their guild through the normal flow.
//
//     This is a player-facing rule change, not only a fix: an officer can no
//     longer add a player by approving on their behalf. Every route into a guild
//     now needs the joining player to transact — request then approve, invite
//     then accept, or direct join.
//
//   - A player id supplied by a message must now name a player that exists.
//     CurrentContext.GetPlayer is a cache allocator, like every other context
//     getter: it never reads the store. It nevertheless returned an error, always
//     nil, and twenty-eight callers guarded on it — guards that read exactly like
//     existence checks and were dead code the compiler could prove unreachable.
//     The error return is gone, which turned every one of those guards into a
//     compile error, and the ids a transaction chooses now go through the new
//     GetExistingPlayer, which loads and returns ErrObjectNotFound (1050).
//
//     MsgAllocationTransfer is where that was live. Its guard on msg.Controller
//     was the only check, CanBeTransferBy asking only whether the caller may
//     transfer, so a transfer to any string committed — the allocation write is
//     keyed by allocation id, so unlike the guild paths there was no empty-key
//     panic to abort it. The result is an allocation controlled by somebody who can
//     never sign for it and a permission row keyed to them: unconnectable, though
//     not stranded, since the source owner keeps PermSourceAllocation on the source
//     and the creator keeps the PermAdmin to transfer it back.
//
//     The other repointed handlers were already safe by accident and change only
//     which error they return, from a permission failure or a not-a-member
//     complaint to ErrObjectNotFound: AddressRegister, GuildUpdateOwnerId,
//     PlanetExplore, PlayerUpdateName / Pfp / PfpCrAttributes / GuildRank,
//     StructBuildInitiate, SubstationPlayerConnect / Disconnect / Migrate, and the
//     guild membership loaders — where the refusal now precedes the write that
//     previously panicked on the empty key and failed the transaction. Two ids the
//     ante and permission resolver read are deliberately not checked: neither
//     mutates, and a phantom owns nothing and holds no permission row, so the
//     permission check that follows refuses it anyway. In the ante that declines a
//     throttle reservation rather than rejecting the transaction, which is the
//     required behaviour there.
//
//     Guarded by TestArch_HandlersResolveMessagePlayerIdsThroughGetExistingPlayer,
//     which reads the handler sources and follows a message field through a local
//     variable, since a loop over a repeated field is one.
//
// State migrations:
//
//   - MigrateGuildJoinBypassLevels: clamp either bypass field to closed on any
//     guild holding a value outside the declared enum. The new default branches
//     already make such a record inert, so this is about the record rather than
//     the exploit: left alone it is a guild permanently closed to joins with no
//     signal as to why, and enough to fail GenesisState.Validate on the next
//     export. Closed is the recoverable direction, since
//     CanUpdateJoinConstraintsBy reads only PermGuildJoinConstraintsUpdate and
//     never the level, so the owner reopens the guild in one transaction.
//     Expected to touch nothing unless a guild was actually attacked.
//
//   - MigrateGuildNameIndex: clear the Guild/name/ prefix and rebuild it from
//     every stored guild name under the pinned normalization. Expected to be a
//     no-op, since the new NormalizeName produces the same bytes as the old one
//     on any Go 1.23/1.24 build; it runs because the index cannot be repaired
//     later (RemoveGuildNameIndex can only reach a row by re-normalizing a name,
//     so a row whose key no longer matches any name is unreachable and would
//     hold that name hostage forever) and because it is the only thing that
//     would clean up state a divergent binary wrote. A guild whose stored name
//     fails the pinned rules, or whose key collides with a guild already placed,
//     has its name cleared rather than carried forward — recoverable with one
//     transaction, where a row that disagrees with the guild's name lets a later
//     rename delete a different guild's row. Both drops log at error level, as
//     does any run that is not a no-op, since neither is reachable from state a
//     correct binary produced. Collisions resolve to the lower guild id, which
//     is deterministic because GetAllGuild walks the prefix in key order.
//
//   - MigrateGuildBankFees: backfill bankConvertInFee / bankConvertOutFee to
//     zero on every stored Guild (nil LegacyDec panic defense).
//
//   - MigrateStructTypes: rewrite every StructType from CreateStructTypeGenesis
//     so canDefend is populated.
//
//   - MigrateDefenderCanDefend: prune defender registrations whose defending
//     struct can no longer defend, emitting the same events as a clear tx.
//
//   - MigrateOreClocksToPlanet: move per-struct ore clocks onto the planet,
//     seed active-quantity counters from online planet-located systems, and
//     clear the old struct attributes.
//
//   - MigrateRaiderArrived: seed blockRaiderArrived on planets with an
//     in-progress raid so the pause-shift math does not treat the whole clock
//     age as paused time.
//
//   - MigrateJailedReactorEnergy: gate every reactor whose validator is already
//     jailed or missing at the upgrade height. These never passed through the new
//     hook, so without the backfill they would keep producing energy forever.
//
//   - MigrateReconcileReactorInfusions: rebuild every reactor infusion from live
//     staking state, clearing the phantom fuel that full redelegations left at
//     source reactors while BeforeDelegationRemoved was a no-op. Runs after
//     MigrateJailedReactorEnergy, so the two agree on a jailed validator's zero
//     ratio and this has the final say on the fuel behind it. Removing capacity
//     no stake backs is deliberately allowed to have consequences: each cleared
//     row queues a grid cascade, and the EndBlocker of the upgrade block sheds
//     allocations until load fits capacity, which can cascade downstream through
//     substations and tear down provider agreements. The counters it logs
//     (infusionsVisited, infusionsChanged, phantomFuelCleared,
//     phantomPowerCleared, playersAffected, commissionDivergence) are the record
//     of how much phantom capacity the chain was carrying. Idempotent, and every
//     write is guarded on the value changing, so a healthy chain emits no events.
//
//   - MigratePrimaryAddressPermissions: grant PermAll to every player's current
//     primary address so the tightened update gate cannot lock out accounts that
//     reduced their own address permissions under the old rule.
//
//   - MigrateFleetQueueLimit: for every planet, seed locationListCount from the
//     live raid queue, leave locationListExtra at 0 (capacity 1), and send every
//     visiting fleet beyond the head home via SetLocationToPlanet.
//
//   - MigrateAgreementCheckpointOverbill: return one block of over-billed revenue
//     per stored agreement from the provider's earnings pool to its collateral
//     pool, clamped to what the earnings pool still holds. Without it every
//     pre-upgrade agreement leaves its provider's pool short by
//     capacity * rate * (1 - providerCancellationPenalty) and the
//     provider-collateral-solvency invariant reports insolvency.
//
//   - MigrateExpireOverdueAgreements: settle every agreement whose end block has
//     already passed at the upgrade height, so the new agreement-expiry-liveness
//     invariant starts clean. Runs after MigrateAgreementCheckpointOverbill so each
//     settlement is paid out of a pool that has had its over-billed revenue
//     returned. Expected to settle nothing on a healthy chain.
//
//   - MigrateStructPhantomAggregates: rebuild every aggregate the destroyed-struct
//     bugs could corrupt from the structs still standing — player structsLoad,
//     per-owner-and-type typeCount, and each planet's planetaryShield,
//     defensiveCannonQuantity, lowOrbitBallisticsInterceptorNetworkQuantity and ore
//     mining/refining active quantities — and clear ready/load/capacity/fuel/power
//     grid rows keyed to structs that no longer exist. Destroyed structs are
//     excluded, since destruction already removed their contributions. Runs after
//     MigrateOreClocksToPlanet, which seeds the same ore counters from the same
//     online structs.
//
//   - MigrateAutoResizeAllocationIndex: clear the auto-resize index and rebuild it
//     from the automated allocations that actually exist, keyed by the source each
//     one names. Rebuilding rather than pruning repairs every way the index can be
//     wrong in one pass: hooks whose allocation was destroyed (the leak this
//     upgrade fixes, and the only one expected on a healthy chain), hooks naming a
//     non-automated allocation, hooks filed under a source their allocation does
//     not claim, and automated allocations missing a hook. Two allocations
//     claiming one source is unreachable through the handlers; if corrupt state
//     has it, the first in store key order wins so every node agrees. Runs last,
//     because anything above it that settles an agreement or sheds load can
//     destroy allocations and the rebuild has to have the final say. Idempotent.
//
//   - MigrateOrphanedAllocationControllers: re-home any allocation whose controller
//     is not a player to the owner of its source object, granting that owner
//     PermAllocationConnection and clearing the phantom's permission row. That is
//     exactly what a legitimate transfer to them would have produced: it does not
//     mint PermAdmin, and it touches no other row, the creator's row on the same
//     allocation carrying the PermDelete that AllocationDelete falls back to. An
//     allocation whose source has no owner in state is logged and left alone rather
//     than guessed at — the source owner can still reach it through
//     PermSourceAllocation on the source. Runs after everything that can destroy an
//     allocation, so it does not hand over one about to be torn down. Idempotent,
//     and a no-op on a chain where no transfer ever named a nonexistent player.
const UpgradeName = "v0.21.0"
