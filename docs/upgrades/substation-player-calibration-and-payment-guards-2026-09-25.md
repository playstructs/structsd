# Player demand calibration and targeted payment guards

September 25, 2026. This follow-up calculates actual loadout demand from the branch
definitions and tests one narrow payment-guard candidate. It does not repeat the
global glut simulation or treat surplus power as inherently undesirable.

**Provisional recommendation:** calibrate ordinary player service around **10 kW
usable per player**, test **0.05–0.06 Alpha per connected player per purchased day**,
and keep outgoing allocation capacity priced separately. A policy decision remains
on whether ordinary service should include the 100 kW World Engine's own draw.
The user has been asked that question; this report shows both alternatives.

## 1. Actual demand from the current definitions

The numbers below include the player's 25 W base load, the command ship where
listed, and all listed structures online. These are **gross draws**, before the
player's own usable primary capacity offsets what the substation must provide.

| Legal loadout profile | Gross draw |
| --- | ---: |
| Player plus command ship | 0.075 kW |
| Command ship, extractor and refinery | 1.075 kW |
| Maximum-draw full fleet, including command ship | 1.915 kW |
| Developed planet and maximum-draw fleet, no generators | 7.715 kW |
| Developed planet and maximum-draw fleet, including Field Generator | **8.215 kW** |
| Same scope including Continental Power Plant, excluding World Engine | 17.465 kW |
| Every struct type present, remaining usable slots filled for maximum draw | 116.515 kW |
| Every planetary type, plus maximum-draw fleet | **116.715 kW** |
| Absolute maximum within the default one-planet/one-fleet slot constraints | 117.215 kW |

The last row can omit some low-draw production types to fit higher-draw structures.
It is an upper bound for this default configuration, not a claim that it is a
desirable loadout or a global bound across arbitrary imported or future game state.

The offline calculation enforces four slots per ambit for each location, allowed
ambits and per-player build limits. The command ship occupies its separate fleet
field, not one of the sixteen ordinary fleet slots. The current planet inventory
has no air-slot types, so those slots are not filled with invented structures.
The optimizer selects legal assignments; it does not simply multiply every type
by four or sum unrestricted copies of unique generators.

These are branch defaults, not empirical average-player measurements. Stored live
type definitions, nondefault slots, future draw changes and actual player loadouts
would need separate verification before a production parameter is finalized.

Sources:
[type definitions](/Users/abstract/Documents/Structs/Code/structs/x/structs/types/genesis_struct_type.go),
[player base draw and starting slots](/Users/abstract/Documents/Structs/Code/structs/x/structs/types/keys.go:333),
[fleet slot and command-ship rules](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/fleet_cache.go:268),
[build limits and build power checks](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/msg_server_struct_build_initiate.go:49).

## 2. Construction does not double the required power

Every one of the 22 current struct types has `BuildDraw == PassiveDraw`. Starting
a build reserves its build draw. Completing it removes that reservation before
adding the running draw. The structure occupies its location during construction,
so it is not an extra free slot on top of a completed full loadout.

Consequently, the legal slot-limited draw envelope also bounds the same layout
under construction. A new large generator still needs its full construction power,
but there is no additional build-plus-running spike for that same struct. See
[InitiateStruct](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/struct_context.go:179)
and [build completion](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/msg_server_struct_build_complete.go:56).

This is a current-definition result, not a restriction on future designs. If a
future upgrade changes build draw independently, the calibration needs updating.

## 3. Ordinary play and large generation are distinct policy targets

A 10 kW usable share covers the 8.215 kW developed Field Generator profile with
about 21.7% headroom. It does **not** cover every generator. The Continental Power
Plant consumes 10 kW by itself; the World Engine consumes 100 kW.

There are two coherent choices:

- **Ordinary-service target:** around 10 kW per player, with large-generation
  operators arranging their additional power. This can come from their own usable
  primary supply or an intentionally more generous substation share.
- **Every-struct target:** around 120 kW per player, covering the entire default
  gross-draw envelope even without personal generation.

Neither target should cap actual delivery. A station can provide more at the same
per-player fee. The target is a calibration reference, not a new operating rating.

Personal generation reduces the net substation requirement. The readiness calculation
combines usable primary capacity with the station's secondary share and compares
that against player and struct load. A gross loadout table is therefore conservative
for players with their own power. It is not safe to assume future generator output
already exists during construction: the bootstrap power still has to come from
somewhere. See [player power accounting](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/player_cache.go:216).

## 4. Translate usable demand into a calibration price

Use the retained 70% baseline and quadratic health decay. Ignore the temporary
bonus when sizing reliable service. With no outgoing commitments, supplying a
usable player target at an age of seven days requires:

```text
raw input per player = usable target / (0.7 × health at seven days)
```

| Usable target | Raw input required at day 7 | Raw input required at day 14 |
| --- | ---: | ---: |
| 8.215 kW | 12.411 kW | 15.003 kW |
| 10 kW | 15.108 kW | 18.263 kW |
| 120 kW | 181.299 kW | 219.156 kW |

These are per-player shares of a station dedicated to direct service. If it also
exports power, outgoing commitments must be deducted before dividing the remainder
among connected players.

Using the earlier candidate bulk marginal rate of 0.0038 Alpha per kW per purchased
day gives these calibration values at seven days of headroom:

- 8.215 kW usable target: **0.04716 Alpha** per player per purchased day.
- 10 kW usable target: **0.05741 Alpha**.
- 120 kW usable target: **0.68894 Alpha**.

At the middle marginal rate of 0.0059 Alpha/kW/day, the 10 kW reference becomes
0.08914 Alpha. These are reference calculations, not exact quotes from the full
multi-tier station tariff; applying a marginal rate to every watt deliberately
omits its earlier brackets and the base fee.

This gives the earlier **0.05 Alpha** test value a plausible ordinary-service basis.
I would test 0.05 and 0.06 rather than claim the model established a single optimal
price. That range is not calibrated to guaranteed 120 kW gross service per player.

Most importantly, **the reference demand sets the fee; actual incoming supply
does not dynamically set this component of the bill.** Surplus capacity remains
useful insurance and is not punished just for exceeding the reference.

## 5. Affordability depends strongly on the base fee and real earning cadence

With the previous illustrative 1 Alpha base and 0.05 Alpha per player, purchasing
one upkeep day every 14 days would cost:

| Connected players | Station purchase | Average cost per player per day |
| --- | ---: | ---: |
| 1 | 1.05 Alpha | 0.075 Alpha |
| 10 | 1.5 Alpha | 0.010714 Alpha |
| 100 | 6 Alpha | 0.004286 Alpha |
| 1,000 | 51 Alpha | 0.003643 Alpha |

That cadence is an affordability scenario, not a guarantee that every station can
maintain service for 14 days. It needs the corresponding input/headroom and remains
subject to supplier changes, attacks and actual commitments.

If a player earns one Alpha every seven days, the 100-player case consumes about
3% of that income. If earnings are one Alpha every thirty days, it consumes about
12.9%. For a solo station at the latter income, it exceeds income. The calculation
also sweeps daily income, four player prices, four cohort sizes and three repair
intervals; these are explicit hypothetical income scenarios, not live earnings data.

**The 1 Alpha base fee needs its own review.** It is a major entry cost for small
stations while barely affecting large guilds. Raising it to make disconnection
tricks unattractive would disproportionately burden those small stations. Energy
requirements alone cannot determine an affordable Alpha price.

## 6. A narrowly tested candidate for the payment bypasses

The previous model showed that an operator could pay while players were disconnected
or dynamic output was small, then restore those services. To test a concrete guard,
I added a **pure pricing calculator**, not a chain implementation:

> Upkeep buys coverage for a number of direct connections and an amount of outgoing
> capacity. Adding service beyond that coverage pays the difference. Leaving and
> returning inside already purchased coverage does not pay twice.

It remembers two numbers: covered connection slots and covered outgoing capacity,
alongside the existing health timestamp and prepaid expiry. These are not player-ID
histories or historical billing logs. They do not cap how much power a directly
connected player can receive.

On a smaller upkeep purchase, coverage resets to the smaller quantities. Therefore
disconnecting players, buying cheap upkeep and reconnecting requires buying back
the coverage that was given up. Dynamic output restoration follows the same rule.

### Targeted arithmetic results

At the previous illustrative service prices:

| Sequence | Total payment with the candidate guard |
| --- | ---: |
| Repair an empty station for one day, then reconnect 100 players | **6 Alpha**, equal to buying for all 100 initially |
| Shrink 60 MW dynamic output to 1 kW, repair with 100 players, restore | **255.36 Alpha**, equal to the ordinary purchase |
| Repair empty for seven days, add 100 players and 60 MW output | **1,787.52 Alpha**, equal to seven ordinary days |
| Players leave and return without a smaller intervening upkeep purchase | **No additional charge** within existing coverage |
| Buy six small days, then expand during a one-day renewal | Expansion also pays for the six carried days |

The candidate's added-service charge is the sum of positive incremental player
and output tariff components, multiplied by the greater of one day or remaining
prepaid time. The one-day floor charges for inherited restored health even after
the temporary bonus expires. Adding service never resets the station's health
clock. A normal new upkeep purchase does.

Raising one component cannot be made free by reducing the other: the two coverage
amounts are tracked separately. Tier differences are computed before rounding, so
many small increases cannot erase charges. The tests include 1,000 randomized
split-growth cases, failed-payment atomicity in the pure calculator, expiry and
seven-day carry-forward behavior.

## 7. What that candidate still requires before adoption

The calculator establishes the arithmetic, **not complete gameplay or chain safety**.
It adds a mechanic even though it requires no manual operating-cap transaction.
I would not describe it as a zero-complexity fix.

**Who pays:** a connection or allocation increase must name an authorized payer and
accept the quoted contribution. An operator cannot silently spend an unrelated
player's wallet; a player cannot silently spend the guild vault. Payer permission,
maximum accepted spend, insufficient funds and existing membership/invite flows
need a deliberate message design. New payment side effects require asset authority
on the signing key, not just connection or allocation authority.

**Automatic output:** an automatic allocation can grow because of a capacity change
outside a normal upkeep transaction. A block hook must not fail because this fee
cannot be collected. One safe candidate is to let automatic output grow within
already purchased coverage and make further growth available through the next
explicit funded upkeep/expansion action. That is a new behavior to approve, not
something the current code or these arithmetic tests already implements.

**Fairness near expiry:** standalone growth inherits old health but pays a one-day
minimum. Close to zero health, a combined repair and expansion may be better value.
The interface would need to show that choice. A more proportional rule would add
arithmetic and should not be introduced merely to optimize a small edge case.

**Every mutation path:** new allocation creation, dynamic increases, provider
capacity increases, player connection/migration, guild joins and any implicit
connection path must apply coverage consistently. Reductions should remain possible
without paying, and teardown must preserve existing settlement guarantees.

These questions are why I have not wired the calculator into keeper code. The
earlier standby rotation problem is closed arithmetically only if all growth paths
actually collect the required contribution; missing one would reopen it.

## Recommendation and next implementation sequence

1. Choose ordinary-service versus every-generator calibration. Keep surplus direct
   capacity unpenalized and avoid recalculating the fee from temporary input levels.
2. Choose the base fee separately from the per-player fee, using affordability for
   small stations as well as large guilds. The evidence supports a playtest range,
   not a final production price or assumed real player income.
3. Review the covered-service candidate's payer and automatic-growth behavior. If
   those changes are unwelcome, explicitly accept snapshot-pricing discounts rather
   than declaring them fixed by connection count alone.
4. Only after selecting those rules, implement message/permission changes and real
   keeper/app regressions for detach/repair/reattach, dynamic shrink/restore, automatic
   output, prepaid growth, vault debits and rollback. Preserve bounded processing and
   accurate events; update `test_chain.sh` as part of that implementation.

No global economic rerun is needed to answer these immediate questions. No new
scheduled upkeep enforcement, full-store scans or permanent financial audit logs
are proposed here.

## Validation and artifacts

All **80 research-model tests passed**, including 11 new calibration/coverage tests.
The loadout calculation verifies slot occupancy, allowed ambits, build limits,
command placement and total draw. It covers 22 definitions and nine legal profiles.
The parameter tables contain 16 capacity targets, 144 affordability scenarios and
four coverage examples. Outputs were replayed and matched byte for byte.

Calculations use the earlier 15,500-block model-day convention for comparability.
At an exact five-second block, a calendar day is 17,280 blocks. Production duration
parameters still need the intended convention. Reported draw is in kW; source
draws are integer mW. Gross demand does not include power the player chooses to
export through allocations, which is accounted for separately.

```sh
python3 -m unittest discover -s tools/substation_model -p 'test_*.py' -v
python3 tools/substation_model/player_energy_calibration.py
python3 tools/substation_model/player_energy_calibration.py --out build/substation-player-calibration-replay
```

- [Calibration script](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/player_energy_calibration.py)
- [Coverage calculator](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/service_coverage_quote.py)
- [Targeted tests](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/test_player_calibration.py)
- [Results, legal loadouts and source fingerprints](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-player-calibration-2026-09-25/results.json)

No production Go, protobuf, permissions, migrations or live-chain state was changed.
