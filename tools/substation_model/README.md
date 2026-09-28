# Adversarial substation economic model

Standalone Python research model for the September 25 design and subsequent user
clarifications. **Not production code, a real chain exploit, or a calibrated forecast.**
These upkeep proposals were superseded by the fixed 10% substation operating
loss. The model sources and reports are retained as research history. Generated
CSV, JSON and chart outputs belong under ignored `build/substation-model/`;
they are local artifacts and can be regenerated with the commands below.
Python 3.10+ standard library is sufficient for the model and tests. Plotting is
optional and uses matplotlib. No original external simulator or live SQL snapshot
was available; the supplied report's aggregate numbers are scenario inputs.

From the repository root:

```sh
python3 -m unittest discover -s tools/substation_model -p 'test_*.py' -v
python3 tools/substation_model/experiments.py
python3 tools/substation_model/plot_results.py build/substation-model/substation-model-results-2026-09-25/results.json
```

Use `--out /path/to/output` to keep another run separate. Use `--parameters` for a
different JSON configuration. The output records the parameter file, random seed,
and SHA-256 of the model, experiment runner and parameter source. Results are
deterministic for the same Python numeric behavior and inputs; this is not a
cross-platform consensus implementation. CSV files accompany the JSON.

## Adopted rules

- Power is integer mW, money integer ualpha, time integer blocks.
- Health is `max(0, 1 - (age / wearBlocks)^2)`; multiply before flooring.
- Active prepaid upkeep removes the 30% baseline penalty; health still declines.
- Outgoing admission uses durable output projected seven days ahead, never the boost.
- Upkeep buys whole days up to seven days ahead and resets health on payment.
- Age alone never enqueues work. Audits or actual supply changes initiate enforcement.
- Fixed allocations shed by numeric creation order; automated output is exclusive
  and survives at zero power. Reductions propagate through a bounded queue.
- Only explicitly accepted inputs contribute to supply. All attacker input cuts in
  scenarios concern previously accepted power, not unsolicited donations.
- Each modeled initial station receives the proposed 0.5 Alpha bootstrap reserve.
- Upkeep burns ualpha; rank-token revenue is not silently converted into ualpha.
- Bounties respect collateral backing and the candidate final-claim dust sweep.

## Assumptions and boundaries

`parameters.json` identifies candidate values. In particular, the reserve, dust
threshold, exact day convention and repair price have not been finalized. The model
has **no development upgrades or automatic generator expiration**, because neither
has a selected rule in the current substation design.

The monetary model assumes existing generators can receive additional infusion at
2/5/10 kW per Alpha; it does not create them without build requirements. Baseline
2 kW results use the least efficient Field Generator case. Flat demand and 3,000
Alpha/day issuance are explicit attack scenarios; there is no assumed market price
or extrapolated human behavior. No combat is assumed in the glut trajectory. The
supply-shock case separately assumes the giant generator has been destroyed; it
does not prove an attacker can win that battle.

The original reported treasury balances and production mix were not independently
reconciled. Treasury experiments use synthetic 29.6 MW supply and 16.3 MW aggregate
demand, not a reconstruction of station 4-1's actual users. Income accrues smoothly
while the modeled cohort can operate, with optional essentials-only load reduction.
Existing member savings, outside sponsorship, token redemption, trading, purchases
of new supply and player departures are excluded from this closed-funding case.
Therefore a cold restart failure is conditional, not an assertion that real players
have no recovery options.

The graph model stores installed allocation credits. It deliberately does not prove
every edge has fresh rooted backing, matching the chosen deferred/player-enforced
model. Audit examples assume a valid proof and solvent agreement settlement. They
summarize local claim reconciliation atomically; they do not simulate bank failures,
provider refunds, payout retries, claim escrow implementation, ante limits or gas.

The cascade budget counts visits and edge operations, not actual Cosmos store gas.
Large-graph timing is illustrative workload at a nominal 5.55-second block interval,
not a chain benchmark. No full-store scanning or dictionary sorting from this Python
model is a recommendation for production implementation.

Audit races separately translate `CalculateDifficulty` exactly into integer math.
Monte Carlo sampling of independent hash arrivals uses floating point off-chain.
Only the supplied small/severe endpoint ranges are used: the proposed interpolation
table was not available. Each race holds that severity fixed. Hashers are continuously
online, have the stated aggregate rate, and face a sniper at difficulty 5. Transaction
inclusion/latency/electricity prices are not modeled. Preparation is permitted only
within the same assumed known upkeep/audit cycle; an intervening anchor change would
invalidate it. Both current state and required difficulty still gate submission.

## Experiments

| Output | Question |
| --- | --- |
| `upkeep_search.csv` | Can owners maintain service cheaply with infrequent purchases? |
| `prepayment.csv` | Do supply expansion and whole-station health resets bypass pricing? |
| `oldest_first.csv` | How much service can a small accepted input withdrawal destroy? |
| `ghost_power.csv` | What happens when an upstream relay ages and nobody audits it? |
| `fragmentation.csv` | How does splitting allocations change drain work? |
| `treasury.csv` | Can lost production prevent the treasury from affording recovery? |
| `glut.csv` | Can permanent generation grow faster than demand despite upkeep? |
| `supply_shock.csv` | What if the giant generator disappears? |
| `audit_races.csv` | How much advantage does preparing work within an upkeep cycle give? |
| `reserve_drain.csv` | How many distinct valid audits exhaust the reward reserve? |

The healthy lossy-cycle negative control is in `results.json`. It does not find
unlimited energy from a settled cycle and drains to zero after external supply is
removed. Repeated reserve-drain rows require a **new real imbalance** each time;
they do not permit replaying the same completed audit.

## Counterfactual pricing safeguards

`naive`: expiry applies to whatever supply exists after purchase.

`capacity_ceiling`: the temporary boost applies only to paid capacity, but payment
still resets health globally. This tests the proposed incomplete fix.

`sticky_billable_ceiling`: price the complete active operating capacity despite a
temporary input reduction. This is a control that closes the demonstrated cheap
reset, not a fully specified dynamic resizing/billing algorithm. Mixed-capacity
stacked purchases are explicitly rejected by this model rather than guessed.

Small invariant tests cover integer price/health oracles, real difficulty inequalities,
collateral conservation, dust boundaries, no age-triggered work, successful zero-health
audits, replay refusal, automated exclusivity, ordered bounded shedding and lossy loops.

## Operating-rating follow-up

`rating_model.py` extends the original grid with a hard operating rating and a
serviced ceiling. It models supply-independent billing, priced expansion, smaller
renewals that forfeit old larger coverage, and paid commissioning of new stations.
The old counterfactual experiments remain unchanged.

```sh
python3 tools/substation_model/rating_experiments.py
```

Outputs default to `build/substation-model/substation-rating-results-2026-09-25`; `--out`
and `--parameters` are supported. The normal test-discovery command includes the
new rating tests. See the [rating report](/Users/abstract/Documents/Structs/Code/structs/docs/upgrades/substation-operating-rating-model-2026-09-25.md)
for exact proposed formulas, candidate-policy assumptions and measured limits.
This remains research code, not a deployed implementation.

## Fixed-price repair alternative

`fixed_repair_model.py` removes the temporary bonus, prepaid duration and capacity
rating. A fixed fee restores health; accepted supply can expand freely. The 70%
baseline remains. Normal creation pays the first repair fee in addition to founding
burn and collateral; explicitly seeded migration fixtures are separate.

```sh
python3 tools/substation_model/fixed_repair_experiments.py
```

Outputs default to `build/substation-model/substation-fixed-repair-results-2026-09-25`.
The runner sweeps repair prices, spare capacity, station partitioning, funding,
replacement, routing and permanent-generation growth. `--out` and `--parameters`
are supported. `plot_fixed_repair.py` optionally produces figures with matplotlib.
The [fixed repair report](/Users/abstract/Documents/Structs/Code/structs/docs/upgrades/substation-fixed-repair-model-2026-09-25.md)
explains the simplification, consolidation incentive, assumptions and limits.
This is an explored alternative, not a finalized fee or chain implementation.

## Delayed supply-based discounts

`discount_delay_model.py` protects the upkeep quote after supply reductions until
the lower level has persisted. It retains whole-station health restoration and
tests a conservative paid-capacity limit on the temporary bonus. Qualification
is lazy; there is no scheduled processing or growing history.

```sh
python3 tools/substation_model/discount_delay_experiments.py
```

Outputs default to `build/substation-model/substation-discount-delay-results-2026-09-25`.
`--out` and `--parameters` are supported. The [delay report](/Users/abstract/Documents/Structs/Code/structs/docs/upgrades/substation-discount-delay-model-2026-09-25.md)
documents the immediate-trick protection, rotating-station counterexample, timer
tradeoffs, exact test scope and unresolved production assumptions.

## Connection-count and outgoing-capacity pricing

`service_pricing_model.py` prices whole-day upkeep using a base fee, a per-connected-
player fee and the existing marginal tariff applied to aggregate outgoing capacity.
It retains health restoration and prepaid benefit, without the discount delay or
manual rating. It tracks unique player connections and distinguishes dynamic
updates from allocation types that cannot be directly resized.

```sh
python3 tools/substation_model/service_pricing_experiments.py
```

Outputs default to `build/substation-model/substation-service-pricing-results-2026-09-25`.
`--out` and `--parameters` are supported. The [service-pricing report](/Users/abstract/Documents/Structs/Code/structs/docs/upgrades/substation-service-pricing-model-2026-09-25.md)
covers the successful input-disconnection protection, deliberate customer movement,
dynamic allocation counterexamples, parameter assumptions and chain-test limits.

## Player demand calibration and service-coverage arithmetic

`player_energy_calibration.py` reads the branch's Go struct definitions and computes
legal default-slot loadouts, reference capacity targets and hypothetical income
comparisons. `service_coverage_quote.py` tests the arithmetic for charging additions
beyond already purchased player/outgoing coverage. It does not implement a payer,
automatic-allocation growth policy or any chain mutation.

```sh
python3 tools/substation_model/player_energy_calibration.py
```

The [calibration report](/Users/abstract/Documents/Structs/Code/structs/docs/upgrades/substation-player-calibration-and-payment-guards-2026-09-25.md)
distinguishes ordinary play from large-generator gross draw, proposes calibration
ranges, and identifies the remaining integration decisions. Results default to
`build/substation-model/substation-player-calibration-2026-09-25`; `--out` is supported.
