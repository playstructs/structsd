"""Fixed-price repair alternative. Research model, not chain implementation."""
from model import ALPHA, Grid, project


class FixedRepairGrid(Grid):
    def __init__(self, p=None, repair_price=10*ALPHA, founding_burn=ALPHA//2):
        super().__init__(p)
        if repair_price <= 0 or founding_burn < 0:
            raise ValueError("invalid prices")
        self.repair_price = repair_price
        self.founding_burn = founding_burn
        self.creation_burned = 0
        self.collateral_funded = 0

    def seed_station(self, name, root=0):
        """Existing/migrated fixture, not a repeatable free creation operation."""
        s = super().add_station(name, root)
        s.last_upkeep = self.height
        return s

    def add_station(self, *args, **kwargs):
        raise ValueError("use seed_station for fixtures or create for paid creation")

    def create(self, name, root, wallet):
        cost = self.repair_price+self.founding_burn+self.p.initial_collateral_ualpha
        if name in self.stations or root < 0 or wallet < cost:
            raise ValueError("invalid creation or insufficient funds")
        s = super().add_station(name, root)
        # Parent fixture helper counts a mint; ordinary creation funds from wallet.
        self.bootstrap_minted -= s.collateral
        self.collateral_funded += s.collateral
        self.creation_burned += self.founding_burn
        s.last_upkeep = self.height
        s.burn = self.repair_price
        return wallet-cost

    def throughput(self, name):
        s = self.stations[name]
        if s.collateral == 0:
            return 0
        return project(self.supply(name), self.height-s.last_upkeep,
                       self.p.durable_bps, self.p)

    def repair(self, name, wallet):
        s = self.stations[name]
        if wallet < self.repair_price or s.collateral == 0:
            raise ValueError("insufficient funds or disrepair reserve missing")
        if s.last_upkeep == self.height:
            raise ValueError("already repaired this block")
        s.last_upkeep = self.height
        s.burn += self.repair_price
        self.events.append({"kind": "repair", "station": name,
                            "height": self.height, "burn_ualpha": self.repair_price})
        # No payment duration, capacity rating or automatic restoration of shed edges.
        return wallet-self.repair_price

    def pay(self, *args, **kwargs):
        raise ValueError("fixed repair has no prepaid days; use repair")


def first_offline_block(supply, demand, p):
    """Exact integer first age at which health-adjusted output cannot meet demand."""
    if demand <= 0:
        raise ValueError("positive demand required")
    if project(supply, 0, p.durable_bps, p) < demand:
        return 0
    lo, hi = 0, p.wear_blocks
    while lo < hi:
        mid = (lo+hi)//2
        if project(supply, mid, p.durable_bps, p) >= demand:
            lo = mid+1
        else:
            hi = mid
    return lo
