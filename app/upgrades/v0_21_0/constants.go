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
// State migrations:
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
const UpgradeName = "v0.21.0"
