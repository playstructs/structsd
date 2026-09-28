"""Integer economic kernel for the proposed, unimplemented upkeep rules.

Power is mW; money is ualpha; time is blocks. Only audit-arrival Monte Carlo
uses floats, outside this kernel. This is intentionally not a Cosmos emulator.
"""
from __future__ import annotations

from collections import deque
from dataclasses import dataclass, field
from pathlib import Path
import json

ALPHA = 1_000_000
KW = 1_000_000
MW = 1_000 * KW
BPS = 10_000


@dataclass(frozen=True)
class Parameters:
    blocks_per_day: int = 15_500
    wear_days: int = 30
    buffer_days: int = 7
    prepay_cap_days: int = 7
    durable_bps: int = 7_000
    tier_boundaries_kw: tuple[int, int] = (100, 10_000)
    tier_rates_ualpha_per_kw_day: tuple[int, int, int] = (9500, 5900, 3800)
    initial_collateral_ualpha: int = 500_000
    dust_threshold_ualpha: int = 10_000
    grid_work_budget: int = 256

    @classmethod
    def read(cls, path: Path) -> "Parameters":
        raw = json.loads(path.read_text())
        return cls(**{k: (tuple(v) if isinstance(v, list) else v)
                      for k, v in raw.items() if k in cls.__dataclass_fields__})

    @property
    def wear_blocks(self) -> int:
        return self.wear_days * self.blocks_per_day


def ceil_div(n: int, d: int) -> int:
    assert n >= 0 and d > 0
    return (n + d - 1) // d


def health_fraction(age: int, p: Parameters) -> tuple[int, int]:
    age = max(0, age)
    square = p.wear_blocks ** 2
    return max(0, square - age ** 2), square


def project(power: int, age: int, efficiency_bps: int, p: Parameters) -> int:
    n, d = health_fraction(age, p)
    return power * efficiency_bps * n // (BPS * d)


def upkeep_price(power: int, days: int, p: Parameters) -> int:
    if power <= 0 or days <= 0:
        raise ValueError("positive paid supply and whole days required")
    a, b = (x * KW for x in p.tier_boundaries_kw)
    r0, r1, r2 = p.tier_rates_ualpha_per_kw_day
    numerator = (min(power, a) * r0 + min(max(0, power-a), b-a) * r1
                 + max(0, power-b) * r2) * days
    return ceil_div(numerator, KW)


def calculate_difficulty(age: int, difficulty_range: int) -> int:
    """Exact translation of x/structs/types/work.go CalculateDifficulty."""
    if age <= 1 or difficulty_range <= 1:
        return 64
    target = age ** 63
    product, exponent = 1, 0
    for k in range(1, 64):
        product *= difficulty_range
        if product > target:
            break
        exponent = k
    return 64-exponent


def bounty(balance: int, shed: int, commitments: int, achieved: int,
           p: Parameters) -> int:
    if balance <= 0 or commitments <= 0 or shed <= 0:
        return 0
    # Piecewise linear 25% at <=5 and 100% at >=9.
    scale = 2500 + min(4, max(0, achieved-5)) * 1875
    result = min(balance, max(1, balance * min(shed, commitments) * scale
                              // (2 * commitments * BPS)))
    return balance if balance-result <= p.dust_threshold_ualpha else result


@dataclass
class Station:
    name: str
    root: int
    collateral: int
    last_upkeep: int = 0
    paid_until: int = 0
    paid_capacity: int = 0
    burn: int = 0
    audit_nonce: int = 0
    outgoing: dict[int, "Allocation"] = field(default_factory=dict)
    incoming: dict[int, "Allocation"] = field(default_factory=dict)


@dataclass
class Allocation:
    index: int
    source: str
    destination: str
    power: int
    automated: bool = False
    accepted: bool = True


class Grid:
    """Local allocation credits plus bounded propagation; no timed work.

    Upstream edge power is an installed claim, not a recursive source proof.
    An audit is modeled conditional on a valid proof; the race experiment
    separately samples probability/difficulty. All settlements are assumed solvent.
    Python dictionary sorting here is model convenience, NOT proposed chain code.
    """
    def __init__(self, p: Parameters | None = None):
        self.p = p or Parameters()
        self.height = 0
        self.stations: dict[str, Station] = {}
        self.next_index = 1
        self.queue: deque[str] = deque()
        self.queued: set[str] = set()
        self.events: list[dict] = []
        self.bounties_paid = 0
        self.bootstrap_minted = 0
        self.prepaid_policy = "capacity_ceiling"

    def add_station(self, name: str, root: int = 0) -> Station:
        assert name not in self.stations and root >= 0
        s = Station(name, root, self.p.initial_collateral_ualpha)
        self.stations[name] = s
        self.bootstrap_minted += s.collateral
        return s

    def supply(self, name: str) -> int:
        s = self.stations[name]
        return s.root + sum(e.power for e in s.incoming.values() if e.accepted)

    def load(self, name: str) -> int:
        return sum(e.power for e in self.stations[name].outgoing.values())

    def throughput(self, name: str) -> int:
        s = self.stations[name]
        if s.collateral == 0:
            return 0
        power = self.supply(name)
        covered = 0
        if self.height < s.paid_until:
            covered = power if self.prepaid_policy == "naive" else min(power, s.paid_capacity)
        n, d = health_fraction(self.height-s.last_upkeep, self.p)
        return (covered*BPS + (power-covered)*self.p.durable_bps)*n // (BPS*d)

    def safe(self, name: str) -> int:
        s = self.stations[name]
        if s.collateral == 0:
            return 0
        return project(self.supply(name), self.height-s.last_upkeep
                       + self.p.buffer_days*self.p.blocks_per_day,
                       self.p.durable_bps, self.p)

    def shared(self, name: str) -> int:
        return max(0, self.throughput(name)-self.load(name))

    def advance(self, height: int) -> None:
        assert height >= self.height
        self.height = height  # NO automatic audit, queue insertion or cascade.

    def pay(self, name: str, days: int, wallet: int, billable: int | None = None) -> int:
        s = self.stations[name]
        until = max(self.height, s.paid_until) + days*self.p.blocks_per_day
        if until > self.height + self.p.prepay_cap_days*self.p.blocks_per_day:
            raise ValueError("prepayment cap")
        power = self.supply(name) if billable is None else billable
        cost = upkeep_price(power, days, self.p)
        if cost > wallet or s.collateral == 0:
            raise ValueError("insufficient liquid ualpha or station in disrepair")
        # Experiments do not mix differently sized stacked purchases. Such
        # stacking needs a selected production policy before this model can certify it.
        if s.paid_until > self.height and s.paid_capacity != power:
            raise ValueError("mixed-capacity stacking deliberately unspecified")
        s.paid_until, s.paid_capacity = until, power
        s.last_upkeep = self.height
        s.burn += cost
        return wallet-cost

    def enqueue(self, name: str) -> None:
        if name not in self.queued:
            self.queue.append(name)
            self.queued.add(name)

    def root_change(self, name: str, power: int) -> None:
        assert power >= 0
        old = self.stations[name].root
        self.stations[name].root = power
        if power < old:
            self.enqueue(name)

    def connect(self, source: str, destination: str, power: int,
                automated: bool = False, accepted: bool = True) -> Allocation:
        s = self.stations[source]
        if source == destination or power <= 0:
            raise ValueError("invalid edge")
        if (automated and s.outgoing) or any(e.automated for e in s.outgoing.values()):
            raise ValueError("exclusive automated output")
        if self.load(source)+power > self.safe(source):
            raise ValueError("admission exceeds durable safe budget")
        e = Allocation(self.next_index, source, destination, power, automated, accepted)
        self.next_index += 1
        s.outgoing[e.index] = e
        self.stations[destination].incoming[e.index] = e
        return e

    def resize(self, e: Allocation, power: int, admission: bool = False) -> None:
        assert power >= 0
        if admission and self.load(e.source)-e.power+power > self.safe(e.source):
            raise ValueError("resize exceeds budget")
        old = e.power
        e.power = power
        self.events.append({"height": self.height, "edge": e.index,
                            "kind": "resize", "from_mw": old, "to_mw": power})
        if e.accepted and power < old:
            self.enqueue(e.destination)

    def destroy(self, e: Allocation) -> None:
        self.events.append({"height": self.height, "edge": e.index,
                            "kind": "destroy", "from_mw": e.power, "to_mw": 0})
        del self.stations[e.source].outgoing[e.index]
        del self.stations[e.destination].incoming[e.index]
        if e.accepted and e.power:
            self.enqueue(e.destination)

    def shed(self, name: str, target: int, allowance: int) -> tuple[int, bool]:
        used = 0
        s = self.stations[name]
        while self.load(name) > target and s.outgoing and used < allowance:
            edge = s.outgoing[min(s.outgoing)]
            if edge.automated:
                self.resize(edge, target)
            else:
                self.destroy(edge)
            used += 1
        return used, self.load(name) <= target

    def cascade(self, budget: int | None = None) -> int:
        left = self.p.grid_work_budget if budget is None else budget
        start, visited = left, set()
        while self.queue and left > 1:
            name = self.queue.popleft()
            self.queued.remove(name)
            if name in visited:
                self.enqueue(name)
                break
            visited.add(name)
            left -= 1  # node visit
            if self.load(name) <= self.throughput(name):
                continue
            edges = self.stations[name].outgoing
            target = self.safe(name) if any(e.automated for e in edges.values()) else self.throughput(name)
            used, done = self.shed(name, target, left)
            left -= used
            if not done:
                self.enqueue(name)
        return start-left

    def drain(self, max_blocks: int = 100_000, budget: int | None = None) -> int:
        blocks = 0
        while self.queue and blocks < max_blocks:
            self.cascade(budget)
            self.advance(self.height+1)
            blocks += 1
        if self.queue:
            raise RuntimeError("cascade failed to drain within bound")
        return blocks

    def audit(self, name: str, achieved: int = 9) -> dict:
        """Conditional valid-proof settlement, not PoW admission emulation.

        Local shedding is summarized atomically for these small fixtures. Real
        multi-block claim/escrow reconciliation is outside this economic kernel.
        """
        old_load = self.load(name)
        if old_load <= self.throughput(name) or self.stations[name].collateral == 0:
            raise ValueError("not eligible")
        _, done = self.shed(name, self.safe(name), len(self.stations[name].outgoing))
        assert done
        shed = old_load-self.load(name)
        s = self.stations[name]
        reward = bounty(s.collateral, shed, old_load, achieved, self.p)
        s.collateral -= reward
        s.audit_nonce += 1
        self.bounties_paid += reward
        if s.collateral == 0:
            self.shed(name, 0, len(s.outgoing))
        result = {"height": self.height, "station": name, "kind": "audit",
                  "shed_mw": shed, "old_load_mw": old_load,
                  "reward_ualpha": reward, "collateral_ualpha": s.collateral}
        self.events.append(result)
        assert self.load(name) <= self.throughput(name)
        return result
