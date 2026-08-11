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
// State migrations:
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
const UpgradeName = "v0.21.0"
