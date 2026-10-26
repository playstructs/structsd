# Operating-capacity rating: model results and proposed rules

September 25, 2026.

**Recommendation: use the rating, with the transition rules below.** It closes the
cheap whole-station health reset in the tested model, supports genuine downsizing,
and needs only constant-size station accounting. It does not prevent permanent
energy abundance or make daily upkeep economically necessary.

This is a standalone economic experiment. No chain implementation, protobuf,
migration or live network state was changed. The original model remains intact.
The user selected rating-based pricing for investigation; the precise top-up,
commissioning and forfeiture rules here are proposed implementation decisions.

## What the model establishes

Using the existing candidate prices:

| Attempt | Result |
| --- | --- |
| Disconnect a 100 MW station down to 1 kW, pay upkeep, reconnect | Still costs **401.36 Alpha** for one day. Supply changes do not change the rating. |
| Lower the rating to 1 kW and buy cheap upkeep | Costs **0.0095 Alpha**, but usable input is capped at **1 kW**. |
| Raise that serviced rating back to 100 MW immediately | Costs **401.3505 Alpha** more; total **401.36 Alpha**. |
| Downsize and restore the rating without paying upkeep | No charge inside already serviced capacity, but health and prepaid expiry do not reset. |
| Buy seven cheap days, immediately expand to 100 MW | Total **2,809.52 Alpha**, exactly seven full-size days. |
| Buy six cheap days, then combine one more day with expansion | Also **2,809.52 Alpha**. Carried prepaid time is not free. |
| Wait until the last block of prepaid coverage before expanding | The one-day minimum restoration charge still applies. |
| Expand a station whose health has reached zero | It still has zero output; expansion does not reset health. |
| Replace a neglected station with a newly created station | The replacement has zero operating capacity until commissioned at the normal upkeep rate. Founding costs and collateral are additional. |
| Split one expansion into many smaller increases | Never cheaper in the tested cases; rounding can make it slightly more expensive. |

Ran **720 attack scenarios**, **13 illustrative cases**, **3 year-long economic
trajectories**, and **572 maintenance-cadence comparisons**. All **31 model tests
passed**: 14 existing tests plus 17 new tests. New checks include 50,000 seeded
random actions, 1,000 split-expansion cases, and 1,000 station-partition cases.

These are bounded tests and arithmetic checks, not a proof that every possible
chain interaction is safe. They establish the pricing behavior of this specified
model; SDK gas, permissions, hooks and bank rollback remain implementation work.

## The small amount of state needed

Store two capacity values, alongside the existing health and prepaid timestamps:

- **Operating rating R:** the maximum input the station can use right now.
- **Serviced ceiling C:** the largest rating covered by its current health cycle.

Always maintain `0 <= R <= C`. The ceiling is one number, not a billing history,
per-purchase ledger, per-allocation bucket or permanently growing lifetime maximum.

Effective input is `min(accepted supply, R)`. Both current output and the durable
outgoing-admission budget use that effective input. Paid upkeep still removes the
baseline 30% penalty, and quadratic health decay still reduces output while paid.

Unused, unaccepted or disconnected supply cannot lower a purchase price. More
supply cannot exceed the rating. A rating can be commissioned before supply arrives,
but health ages from the actual upkeep payment; connecting later cannot refresh it.

## Exact candidate pricing

Let `T(x)` be the existing marginal tier tariff for one day at rating `x`, before
rounding. Use fixed-point/integer arithmetic and round the final charge upward to
ualpha. Let `remaining` be the nonnegative prepaid duration in block-days.

### Change the rating without resetting health

For a requested rating `N`:

```text
extra_daily_tariff = max(0, T(N) - T(C))
charge = ceil(extra_daily_tariff * max(1 day, remaining))
R = N
C = max(C, N)
health timestamp and prepaid expiry remain unchanged
```

Moving inside an already serviced ceiling is free and provides no new health or
time. Expanding above it pays for the added capacity. Use the difference between
total tariffs, not `T(N - C)`, so expansion continues through the existing marginal
price bands. Compute the difference before rounding to prevent free dust steps.

The **one-day floor** is a deliberately simple minimum service charge for new
capacity. Without it, expansion after prepaid expiry would cost zero while that
new capacity inherited the station's remaining restored health. It conservatively
charges the same floor even when little health remains. In that situation the UI
should recommend an upkeep purchase at the new rating instead.

### Buy upkeep, optionally changing rating in the same transaction

For `d` whole purchased days at target rating `N`:

```text
charge = ceil(T(N) * d)
       + ceil(max(0, T(N) - T(C)) * remaining)
new expiry = max(current height, old expiry) + d days
require new expiry <= current height + 7 days
R = C = N
health timestamp = current height
```

The second term covers expansion of **prepaid time already carried forward**.
Without it, buying six days at 1 kW and one day at 100 MW grants seven days of
large-station coverage for approximately one large-station day.

Combining expansion and upkeep avoids charging standalone expansion followed by
another full restoration fee. If prepaid coverage has expired, the combined
purchase is simply the normal full tariff at the new rating.

### Downsizing is real, and lower-price renewal forfeits old coverage

Lowering `R` immediately lowers usable capacity and queues the existing cascade
when necessary. It grants no refund. Before another upkeep purchase, increasing
back to `C` reuses the old health and expiry.

**Once upkeep is purchased at a smaller rating, `C` becomes that smaller rating.**
All old larger-capacity coverage is forfeited, including unexpired benefit. Restoring
the larger rating must then purchase its coverage again. This is what prevents the
downsize–cheap reset–upsize loop without keeping separate health tracks.

This can be expensive if the owner discards prepaid large-station service and
immediately buys it back. The model's deliberately wasteful six-day large purchase,
small one-day renewal, and immediate restoration costs **5,217.623 Alpha**. An
ordinary seven-day large purchase costs **2,809.52 Alpha**. Show that forfeiture
clearly before confirming a lower-rating purchase; do not silently refund it.

## Creation and migration need different initialization

A newly created station must start with `R = C = 0`, or pay the equivalent
commissioning charge as part of creation. It cannot receive unlimited fresh
operating capacity for the small flat founding fee. Otherwise owners could replace
stations instead of paying upkeep, defeating this fix.

At the candidate rates, commissioning 1 MW for one day costs **6.26 Alpha**;
100 MW costs **401.36 Alpha**, in addition to creation costs and collateral. This
is a material initial cost and should be included in the gameplay design.

Existing stations need a one-time migration policy. The model seeds their rating
and serviced ceiling from their initial accepted supply, then applies normal rules
to subsequent purchases. That initial service is a migration concession. The
previously agreed collateral mint remains separate: collateral does not pay upkeep.
The exact migration initialization should be approved with the final design;
never expose the model's migration seeding operation as a normal creation path.

The model gives fixtures their collateral without charging a founder. This isolates
upkeep pricing; it is **not** a proposal to mint collateral for repeated new stations.

## What remains unchanged economically

The rating closes underpayment for restored capacity. It does not change the value
of a correctly priced restoration lasting weeks.

The cheapest full-service cadence in the tested grid still uses **32.6 MW** of
supply and rating for **16.3 MW** demand, buying one upkeep day every **16 days**.
It costs **145.24 Alpha per purchase**, or **9.0775 Alpha/day**, and meets the
modeled demand throughout the cycle. That remains about **0.303%** of the assumed
3,000 Alpha/day issuance. This is a legitimate strategy under the current rules.

The annual glut scenarios also reproduce the earlier results when expansion is
paid atomically with normal upkeep:

| Issuance invested in permanent generation | Available capacity / fixed demand at day 365 | Annual upkeep burn |
| --- | ---: | ---: |
| 0% | 6.60× | 16,366.86 Alpha |
| 10% | 15.97× | 27,570.78 Alpha |
| 50% | 53.45× | 72,386.46 Alpha |

These assume no combat, fixed demand, the supplied aggregate initial state, 2 kW
per Alpha added to existing generators, and one upkeep day every 14 days. They
are economic scenarios, not a price forecast or evidence of unlimited generation.

No age-triggered schedule is added. Passing time alone never queues shutdown;
player audits still enforce neglect. Actual rating reductions use the existing
bounded cascade, including exclusive automated-output resizing and oldest-first
ordinary shedding. Treasury shortfalls and unchallenged stale downstream claims
remain separate properties of the broader design.

## Implementation implications

1. Add rating and serviced-ceiling state, explicit initialization and query fields.
   Validate bounds and use overflow-safe deterministic arithmetic.
2. Make rating the ceiling for every output, shared-capacity and durable-admission
   calculation. Acceptance of supply must not silently buy a higher rating.
3. Offer an atomic upkeep purchase with an optional target rating and maximum
   spend. Use authorized liquid ualpha, proper asset permissions and cached writes.
4. Treat a rating reduction as a capacity mutation: preserve agreement settlement
   ordering, queue bounded propagation, and update automated outputs consistently.
5. Emit accurate old/new ratings, serviced ceiling, amount burned, health anchor
   and prepaid expiry. Existing payout, address and supply events remain required.
6. Reproduce these transition tests with real messages, failed bank payments,
   transaction rollback, multiple messages in one transaction, and measured gas.
   Test creation/replacement and migration independently.

Pricing itself requires only the station state and a fixed number of tariff tiers.
It introduces no growing history or graph traversal. Grid mutations still have
their existing bounded processing obligations; the Python model is not a gas test.

## Reproduction and artifacts

```sh
python3 -m unittest discover -s tools/substation_model -p 'test_*.py' -v
python3 tools/substation_model/rating_experiments.py
python3 tools/substation_model/rating_experiments.py --out build/substation-rating-replay
```

- [Rating model](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/rating_model.py)
- [Adversarial tests](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/test_rating_model.py)
- [Experiment runner](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/rating_experiments.py)
- [Results, parameters and source fingerprints](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-rating-results-2026-09-25/results.json)

CSV files alongside the JSON contain the individual cases. A second run was checked
for byte-identical output. Candidate prices and durations remain configurable.
