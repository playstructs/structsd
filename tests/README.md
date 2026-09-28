# Structs integration tests

Bash scripts that broadcast real transactions against a live `structsd` chain. They are separate
from the Go test suite (`make test`), which needs no chain and no daemon.

Prerequisites for all scripts: a chain running locally, with `alice` (genesis validator) and
`bob` (faucet) in the keyring. Scripts assume home `~/.structs` and `--keyring-backend test`.

Not safe to run concurrently — one test runner per chain. Each script exits non-zero if an
assertion fails.

## test_chain.sh

Full lifecycle: player setup, guild membership, planet exploration, struct building, mining and
refining, fleet movement and combat. A fresh chain is required unless using
`--resume-from`. Before writing fixtures, the script checks for no structs,
planets, guilds, substations or allocations and at most one initial player.
These bounded queries share one committed height; failed, malformed or partial
responses refuse startup rather than counting as empty state.

`make test-integration` runs this with no flags, which includes the proof-of-work phases and
takes hours. Invoke it directly when you want flags:

```bash
bash tests/test_chain.sh --skip-mining
```

- `--skip-mining` — skip ore mining, refinery build/refine and mining-dependent checks.
  Raid command-state scenarios still run. This avoids the slowest ore proof-of-work operations.
- `--extended-battle` — build all 13 fleet struct types and exercise every combat mechanic.
- `--log-battle` — write confirmed typed `EventAttack` details as JSONL under `tests/battle_logs/`. Rejected deliveries and unverified attempts remain separate records and do not increase the event count. A missing or invalid receipt/event fails log verification.
- `--resume-from N` — skip to phase N, recovering IDs by querying the running chain.

Phase 10 requires a live Ore Extractor and Ore Refinery when mining is enabled.
It restores both Command Ships and the required fleet locations, restores the
ore roles, and verifies the expected planet, owner, raider and active
ore clocks, then requires each compute command's specific raid-pause error.
Command recovery follows the fleet's command association and preserves its
current land/water/air/space ambit. A missing command is rebuilt on land; a live
command in another ambit must not be mistaken for a missing ship.
Unavailable fixtures, failed queries and unrelated errors fail coverage. After
the raid, both ore clocks must advance and the raider-arrival marker must clear.
Only `--skip-mining` opts out of these checks. Offline controls in
`test_chain_raid_pause.py` execute the actual shell blocks and transaction-result
helpers; they do not replace native phase execution.

Shield refusals must match `shields_active` or the CLI's absent-window error.
The successful raid uses a common-height baseline and post-state at its delivered
receipt height. `raid_result.py` requires exactly one matching theft, successful
raid and proof event; verifies all stolen ore reached the raider; preserves buried
planet ore; and checks fleet recall, cleared queue links and exact clock shifts.
Zero-loot raids remain checked with `--skip-mining`; the mining-enabled fixture
must have ore to steal. A valid-looking response from a failed query is rejected.

Phase 11 restores both commands and Player 2's Tank on direct resume. The
Command Ship is moved to land for the Tank's weapon; a command with fewer than
three health is replaced so it can survive and counter. The live catalog,
confirmed receipt and state at its committed height must show two volley damage
and two counter damage. Tank armor does not reduce counters in the current
implementation. A wounded Tank may be destroyed by that counter; the command
must survive. Missing roles, failed queries and unrelated errors fail coverage.

Phase 12 reserves missing ships using current free slots and reuses live or
pending roles. It still initiates deferred builds before completing the ships
needed immediately. The two Battleships and two Tank roles must be distinct;
the armor-piercing target is restored to full health, and Player 2's Battleship
stays unbuilt for Phase 13. Batch deactivation requires a delivered missing-object
rejection with both ships unchanged, then exact status events and height-pinned
state for deactivation and each reactivation. A query failure cannot masquerade
as an offline ship.

RG2 (relocated-defender regression) requires its combat roles. It reuses live
structs, completes pending builds, activates offline structs or rebuilds missing
ones. No free slot or a failed prerequisite fails the phase. Its assertions require
the stale defender registration, a confirmed attack receipt and struct/fleet
state at explicit heights; an unavailable fixture cannot count as coverage.

The evasion phase likewise requires all eight scheduled attacks. It validates the
live weapon/defense configuration, clears registered defenders that would obscure
unit evasion, and checks exact damage, target counters and resulting state against
each confirmed receipt. It accepts either random result and reports the observed
counts without testing a probability from eight attempts. Shared recovery moves
a fleet home before rebuilding and returns it afterward; command ships wait for
their separate command slot instead of taking an ordinary ambit slot.

The attack-run phase requires all 15 cases. `attack_run.sh` restores six distinct
fighter roles and each case's target/defenders as needed; `attack_run.py` verifies
the live catalog, actual defender registrations, projectile/counter receipts and
canonical state. The matrix includes nonparticipating Mobile Artillery/Destroyer
registrations, genuine SAM/Submersible cross-ambit counters, Battleship blocks,
lethal counters that prevent a volley, and both visiting-fleet directions.
Destroyed fixtures cannot silently remove a case. Slot selection rejects failed
or inconsistent queries and waits only when a valid occupancy array is full.
Counter/block cases also require enough attacker health to reach their intended
path. A damaged fighter is removed by an independently checked ordinary command
ship attack before rebuilding, so spare hulls do not fill every slot. Preparation
receipts are recorded separately and do not count toward the 15 cases.

Phase 13 restores its full matchup on direct resume and requires four confirmed
results: unbuilt-target refusal, primary-weapon range refusal, armor-piercing Tank
damage, and the defended Command Ship attack. The SAM, Submersible and both
Battleships can all counter into space. Their first three counters in byte-sorted
struct-ID order must destroy the full-health attacker before its volley; the
fourth defender, blocker and target must cause no later damage. Every result is
checked against committed health and exact event identities. The unbuilt target
and a wounded armor target are recreated when necessary on a later resume.

Phase 14 requires all three Cruiser-versus-Interceptor attacks, including one
with the Command Ship offline. It restores destroyed targets and checks the
delivered attack event against health at explicit heights. The offline case
also requires the same living Command Ship to remain offline and unchanged.
Defense set/clear checks both sides of the relationship; stealth toggles must
change the Bomber's visible state. These fixtures are restored on phase-14
resume, so missing earlier builds cannot silently remove coverage.

Phase 13b also restores its Command Ship, SAM and Stealth Bomber on direct
resume. It establishes a defense assignment before clearing it, checks both
the protected list and defender's reverse index, and verifies the restored
assignment. A resumed hidden Bomber is revealed before testing both stealth
transitions. Failed queries, missing objects and inconsistent attribute queries
fail the phase; a missing Bomber cannot skip it.

Phase 15 restores both commands, a Field Generator and its attacking Tank. It
finishes any pending build/activation before taking a height-pinned baseline
from the generator, player, type catalog and infusion inventory. The fixture
requires zero or one position belonging to Player 4's vault at rate two with
zero commission; unexpected owners, extra positions or incomplete pagination
fail before its top-up. Pending restoration gets a bounded wait. Existing fuel
is included in the destruction expectation, so a resumed funded generator must
lose both its old output and the newly purchased output when destroyed.

Phases 17–17c recover the five dedicated fleet players without reallocating power
already committed to their feeder or re-exploring an existing planet. An
interrupted allocation creation can finish its connection. Missing/offline
commands are rebuilt/activated; fleets return home before queue checks and
commands share the land ambit for combat. Queue refusals must identify
`queue_full`. Both attack directions are mandatory, with exact delivered damage,
surviving-target counter damage and historical health/destruction checks. A
counter-killed command is restored before the next scenario, and the final
visitor kill must recall its fleet and release the queue slot. These phases also
support direct `--resume-from 17b` or `17c` after earlier combat attrition.

RG1 (defender counter kills the attacker before a volley) is also required.
It restores a non-evading Starfighter target and a distinct same-ambit Battleship
defender. If the attacker lacks the required one health, it replaces a damaged
fighter when necessary and uses a verified guided attack to reduce a fresh
fighter from three health to one. The preparation accounts for the target counter
too. Both preparation and regression require exact receipts and canonical state
at the confirmed height; neither missing fixtures nor prior battle attrition can
skip the regression. Defender registration and the visiting fleet queue are
checked before the attack, and the registration is cleared afterward.

Extended-battle case B6 (Frigate in space attacking a Pursuit Fighter in air)
restores both roles and the fleet position after earlier combat. It uses the same
confirmed evasion/state checks, with a separate extended-battle outcome and no
contribution to the eight EV cases. The catalog must allow the Frigate to target
air; the Pursuit Fighter's air-only weapons must never counter the space attacker.
Command recovery updates both P3 command aliases used by later script phases.

The dead-bomber ordering case C1b is required. It restores the bomber, Mobile
Artillery target, Tank blocker and High Altitude Interceptor counter. The Tank
must sort before the interceptor by the actual byte order of struct IDs. If it
does not, bounded, verified combat retires and replaces defenders, including the
case where decimal ID width changes. The bomber is revealed and prepared at
exactly one health through a verified interceptor hit; a two-health bomber is
retired before replacement. The regression must identify the lethal interceptor
counter, report no block or projectile damage, and preserve the target and both
defenders' committed health. Setup attacks do not count as regression attempts.

The remaining group C checks are required too. C2 preserves the Battleship versus
Battleship matchup with a Frigate and Starfighter defending. Every attempt checks
exact counters, evasion, block damage and committed health. It retries after
attrition until a block is observed, and fails after 32 verified attempts without
one. Evasion and an attacker killed before firing never count as blocking.

C3 requires exact Tank armor reduction. C4 requires two confirmed hits taking one
full-health Starfighter through destruction, replacing a counter-killed attacker
between shots when necessary. C5 requires an actual delivered off-ambit refusal,
unchanged combat state, and a same-ambit Command Ship hit; command defeat must
return the fleet home, after which the fixture restores its command and battle
position. These checks can rebuild units and therefore add proof-of-work time.

Group E also requires its full planetary cannon sequence. A Battleship damages
the extractor and takes cannon fire; Mobile Artillery hits the same extractor
without taking cannon fire; a Tank attacks a fleet target without triggering the
cannon. Verified artillery hits soften the cannon to two health, then the Tank
kills it without retaliation. That delivery must remove one cannon and thirteen
shield strength. Finally, a Battleship destroys the same extractor with no cannon
damage. Every attack checks its receipt and delivery-height unit and planet state.
Wounded planet fixtures are retired through verified combat and rebuilt when
needed; missing fixtures and failed queries are failures, never skips. Repeating
this group rebuilds the cannon and extractor and can take several minutes.

JS1 requires three delivered unguided Battleship hits against a planet-category
Ore Extractor on the owner planet of an active Jamming Satellite. It verifies
the satellite, the exact one-third interception rate, each attack receipt and
the delivery-height health and planet state. It restores lost roles between
shots and reuses an existing satellite rather than exceeding its build limit.
Fleet targets never substitute for the extractor. The deterministic Go test
separately proves guided jamming, unguided bypass and the fleet exclusion with
an interception rate of one.

## ante_regression.sh

Replays the incident 2026-05 shape: a single transaction carrying two `MsgStructDefenseSet`
messages from the same registered player. Asserts the broadcast is rejected with `structs-ante`
code 2020 (`ErrDuplicateChargeInTx`) — a success is a regression.

Needs a player with a built combat struct and two protectable structs, which phase 1+ of
`test_chain.sh` produces:

```bash
PLAYER_KEY=player_3 \
DEFENDER_STRUCT_ID=5-1702 \
PROTECTED_STRUCT_ID_1=5-1657 \
PROTECTED_STRUCT_ID_2=5-1658 \
make test-ante-regression
```

Optionally set `NODE` (default `tcp://localhost:26657`) and `CHAIN_ID` (default
`structslocal-1`). Exits 0 when the fix holds, 1 on regression, 2 on a missing prerequisite.

## test_upgrade.sh

The runner targets v0.21.0 → v0.22.0. Earlier native upgrade and cold replay
results predate the September 23 reactor replacement and do not certify its
candidate. Re-run the current gates
and frozen evidence in the [v0.22.0 test audit](../docs/upgrades/v0.22.0-test-chain-audit.md).
Offline preparation is available through
`build_upgrade_binaries.sh` and
`python3 -B tests/prepare_upgrade_rehearsal.py --output-parent /existing/directory`.
The latter preserves the released fixture and prepares isolated CLI routing;
it does not initialize a chain or run transactions. See the
[v0.22.0 test audit](../docs/upgrades/v0.22.0-test-chain-audit.md).
Adding `--bootstrap` creates test keys and a validated offline genesis in the
new rehearsal directory. It refuses existing homes and still does not start a
node or broadcast transactions.

The live runner requires a fresh bootstrapped artifact. It populates release-era
state, confirms the governance proposal and vote, captures the upgrade boundary,
swaps binaries, captures block events and compares staking exports. It stops only
its own processes, refuses occupied default ports, and checks for any active
native Structs node immediately before startup, including custom-port nodes.
It preserves logs/exports.
Population failure aborts before governance. Event associations supply the vault
map for staking verification. These checks do not constitute release approval:
The runner also replays all-denomination bank events against the boundary exports
and checks that associated source balances reached their owner's vault. It rejects
unexpected transactions in that comparison window. Vesting events must also
match complete exported vault accounts and source identity conversions, preserve
the original source schedules and reconstruct ordinary delegated backing.
Infusion records and effective grid values must also match event replay.
`diversity-verification.json` separately checks the exact owner bonus from
commission-rounded infusion shares, aggregation across addresses, generator
exclusion, clipping and owner capacity, then compares owner-summary events.
The native export/restart runner applies the same independent check and requires
one first-block reset before replayed summaries. Pending native backing remains
a separate check; an old checkpoint whose unbondings mature during replay cannot
satisfy the exact pending-entry migration comparison.

For full fresh gameplay acceptance, run both modes separately:

```bash
python3 -B tests/run_full_lifecycle.py --binary /path/to/candidate/structsd \
  --output-root /existing/evidence-directory --skip-mining
python3 -B tests/run_full_lifecycle.py --binary /path/to/candidate/structsd \
  --output-root /existing/evidence-directory
```

Each invocation freezes the current source and binary, creates fresh local
`alice`/`bob` keys, validates genesis, runs extended battles and battle logging,
exports the result, then reopens the database for three more blocks. Source,
configuration and binary hashes are retained beside logs and process status.
It refuses other native nodes and occupied rehearsal ports; stop local Docker
nodes separately and restore them afterward. It never stops unrelated processes.
The wrapper pins home, chain ID and RPC for every network command. Local gRPC
must remain enabled because SDK startup otherwise omits simulation query routing.
Fast block/test timings affect only fixture timing, not economic or PoW defaults.
A script failure preserves the failed fixture and does not count as acceptance.

For a nonzero-bonus native lifecycle after a successful isolated upgrade, run:

```bash
python3 -B tests/run_diversity_lifecycle.py /path/to/verified-rehearsal \
  --binary /path/to/candidate/structsd --output-root /existing/evidence-directory
```

This requires the local `alice` and `bob` fixture keys, copies the stopped home,
creates a small second validator, and checks vault infusion, defusion,
cancellation and native database reopening. It saves height-pinned paginated
query evidence and transaction receipts. It refuses occupied rehearsal ports.
The second validator is not started; the fixture's original validator retains
enough voting power for local consensus. It neither alters the source home nor
broadcasts to a remote chain. Run `run_export_restart.py` on its resulting `chain`
directory for both genesis import modes with nonzero diversity.
For signed native cap refusal, failed required guild migration, downtime and
sticky activation across database reopening, run a separate fresh fixture:

```bash
python3 -B tests/run_reactor_cap_lifecycle.py --binary /path/to/candidate/structsd \
  --output-root /existing/evidence-directory
```

This fixture explicitly lowers only the cap's activation threshold to two
validators and the downtime window to 1,000 blocks. Cap percentage and diversity
parameters retain their production defaults. Its two offline additional validators
together have less than one-third of voting power, so the local primary can keep
committing. The owner spreads stake across all three reactors.
Rejection assertions use signed, fixed-gas transactions and exact committed error
codes rather than simulation failures. Successful native outflow, failed native
and vault deposits, failed cancellation, guild membership/stake rollback, the
same player joining by enabled rank purchase without moving stake, bonus loss
on native downtime, persistent activation and cold reopening are checked.
It freezes source/binary/configuration, saves query and transaction receipts, and
uses the same serial-node/port rules as the other runners. `--prepare-only`
creates the offline fixture without starting a node.

Allocation replacements/deletions must reconstruct the full exported inventory;
each legacy undeclared mode requires exactly one replacement to static, while
declared modes remain unchanged. The runner saves `allocation-verification.json`
and refuses missing or inconsistent migration events. Final
release acceptance still requires the remaining vesting cases, native queue
completion/spendability, ordinary agreement payments, economic/queue invariants,
production cardinality, chain-event accuracy/delivery and the full audit.
Indexer implementation and downstream ingestion testing are separate user-owned work.

```bash
bash tests/test_upgrade.sh /path/to/fresh-bootstrapped-rehearsal
```

- `--new-binary` selects the v0.22.0 executable; default `build/structsd-v0.22.0`.
- `--population-timeout` bounds the release fixture run in seconds (default 7200).
- `--vesting-proof-tool /path/to/legacyvestingproof` adds all four SDK vesting
  account types, an unassociated delayed control and ordinary active/pending
  stake. `--vesting-only` selects that focused fixture without repeating gameplay.
- `--live-agreements` creates three confirmed legacy contracts sharing a fresh
  provider pool after the full released fixture has finished its own teardown.
  The contracts use rate 3 ualpha, capacities 2/1/3 and durations 800/810/820
  blocks, with provider/consumer cancellation penalties 0.5/0.25. Receipts and
  identities are saved in `agreement-seed.json`; ambiguous identities or altered
  terms abort before governance. This flag requires fresh full population.
  The upstream source is the fixture player's current, player-owned substation:
  the released script already allocates all of Alice's player capacity there.
  The helper resolves and checks ownership before sending any seed transaction.
  Whenever the boundary has live agreements whose expiry follows the comparison
  window, the runner also verifies unchanged terms, allocation power, provider
  clocks/load and both pool balances, including during cold replay. It saves
  `provider-accounts.json` and `agreement-verification.json`. Later expiry and
  settlement remain separate continuation checks.
- `--replay-checkpoint` treats the directory argument as a stopped
  `upgrade-checkpoint`, creates a private clone and runs the candidate there.
- Rebuild versioned binaries after code changes. Each rehearsal needs a fresh
  prepared artifact; the runner does not reuse or delete an existing chain home.
- Starting the live runner requires the authorization described in `AGENTS.md`.

`continue_upgrade_vesting.py` exercises a dedicated short fixture after its
original upgrade and matching cold replay have both passed. It requires a
1,800-second schedule setting and a 600-second unbonding period prepared in the
private genesis before startup. The ordinary source must contribute 800,000
ualpha of delegated backing and two distinct 100,000-ualpha unbondings must remain
pending after migration. Perform the cold replay immediately, then start
continuation on the original home before the first periodic installment:

```bash
python3 -B tests/continue_upgrade_vesting.py /path/to/short-rehearsal \
  --replay /path/to/short-rehearsal/structs-upgrade-replay-CHILD \
  --new-binary /path/to/frozen/structsd-candidate
```

The driver records each observation at one explicit height, compares native bank
spendability with an independent per-source vesting model, observes both queue
completions separately, locates each first mature block by its exact timestamp,
and requires matching native completion, vault-credit and unbonded-pool-debit
events. It also requires delivered allowed/refused transfers and reopens
the database for continued operation. Evidence goes to an exclusive
`vesting-continuation` directory. A missed boundary or incomplete run fails;
existing evidence is preserved. The driver is covered by offline controls;
successful native execution of the new mixed short fixture remains outstanding.

For the combined `--live-agreements` fixture, keep pruning disabled in the private
`config/app.toml` from before startup so historical export heights survive.
After vesting continuation has passed and stopped its node, verify the three
legacy contracts through expiry:

```bash
python3 -B tests/run_upgrade_agreement_expiry.py /path/to/short-rehearsal \
  --new-binary /path/to/frozen/structsd-candidate
```

This driver briefly reopens the original home to collect complete historical
`block_results`, then stops it before exporting the selected heights. It rejects
any transaction in the interval, changed service terms/load/checkpoints or earlier
provider payments, and checks exact aggregate earnings, individual holdbacks,
residual collateral and per-block deletion/payment events. The interval lies
between the continuation's initial sends and its later timed-unlock transfers.
The original upgrade/cold-replay proof remains separate. Full RPC envelopes,
exports, hashes and failures are preserved under an exclusive `agreement-expiry`
directory. This is healthy ordinary expiry evidence; it does not cover outages
or repricing. A completed continuation and unchanged frozen binary are required;
the driver never starts alongside another native node or sends transactions.

## Native genesis export/restart

After the source local chain is stopped, run both export modes sequentially:

```bash
python3 -B tests/run_export_restart.py /path/to/stopped/chain \
  --binary /path/to/frozen/structsd \
  --output-root /path/to/existing/evidence-directory
```

`--preflight-only` checks inputs and the serial-run guard without exporting or
starting a node. The driver refuses an active native Structs node, including one
using custom ports. It exports the source at preserved height and zero height,
creates separate private homes with distinct local chain IDs, imports each
genesis and commits at least three blocks. It uses localhost ports 29656/29657,
disables peers and API/gRPC servers, and copies only the local validator key into
each freshly initialized home. Source databases and signing state are not copied
or reset. No transactions are sent. The source validator must belong to the
exported set so the isolated test node can produce blocks.

For each mode, the driver saves the original export, imported genesis, public
block results, a new export at the exact observed height and a verification
report. `verify_export_restart.py` checks every bank denomination against public
debit/credit events, reconciles supply against mint/burn events and bank totals,
and checks auth identities, vault owners, membership/ranks, guild settings,
signing associations/nonces, activation and the staking operation-ID counter.
It does not assert all gameplay/staking state or independently prove zero-height
preparation: those retain their real-SDK and lifecycle tests. A prepared driver
or passing offline verifier is not evidence that a native restart has run.

## test_chain_v0.15.0.sh

Historical frozen copy of `test_chain.sh` as it stood at v0.15.0. It is not used
by the current upgrade runner; don't run it against a current chain.

### Ordinary payment checks after billing-history removal

Phase 16 uses `agreement_terms.py` to price confirmed open/resize/extension/close
receipts with integer arithmetic. Resize remainders remain in collateral until
provider deletion. Phase 16b checks cancellation priority and expiry. Phase 16c
uses `provider_lifecycle_accounting.py` for two live contracts, a lazy checkpoint
withdrawal, then immediate consumer settlement and final pool draining. There is
no polling of historical claims or background billing workers.

The quiet-upgrade and expiry oracles in `verify_upgrade_agreements.py` and
`verify_upgrade_agreement_expiry.py` preserve contract start heights and provider
clocks through migration, then check aggregate checkpoint rounding and individual
holdback rounding against exact block receipts and pool balances. The quiet
migration fixture must finish before any expiry; outages, concurrent transactions
and repricing require separate fixtures. Passing Python fixtures is not evidence
that a final native upgrade/lifecycle rehearsal has run.
