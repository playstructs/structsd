"""Candidate operating-rating rules. Research only; no consensus implementation.

Keeps the original counterfactual model intact. All money arithmetic is integer.
"""
from dataclasses import dataclass

from model import BPS, KW, Grid, Station, ceil_div, project, upkeep_price


def tariff_numerator(power, p):
    """Exact daily tariff in ualpha * KW; do not round each tariff first."""
    if power < 0:
        raise ValueError("negative capacity")
    a, b = (v * KW for v in p.tier_boundaries_kw)
    x, y, z = p.tier_rates_ualpha_per_kw_day
    return min(power, a)*x + min(max(power-a, 0), b-a)*y + max(power-b, 0)*z


@dataclass
class RatedStation(Station):
    rating: int = 0
    serviced_ceiling: int = 0


class RatingGrid(Grid):
    def add_station(self, name, root=0):
        # The inherited bootstrap collateral is a test fixture, NOT repeatable
        # production minting. New stations have no serviced operating capacity.
        if name in self.stations or root < 0:
            raise ValueError("invalid station")
        s = RatedStation(name, root, self.p.initial_collateral_ualpha)
        self.stations[name] = s
        self.bootstrap_minted += s.collateral
        return s

    def seed_station(self, name, root, rating=None):
        """Explicit existing/migrated fixture; never a normal creation path."""
        s = self.add_station(name, root)
        s.rating = s.serviced_ceiling = root if rating is None else rating
        return s

    def effective_supply(self, name):
        return min(self.supply(name), self.stations[name].rating)

    def throughput(self, name):
        s = self.stations[name]
        if s.collateral == 0:
            return 0
        efficiency = BPS if self.height < s.paid_until else self.p.durable_bps
        return project(self.effective_supply(name), self.height-s.last_upkeep,
                       efficiency, self.p)

    def safe(self, name):
        s = self.stations[name]
        if s.collateral == 0:
            return 0
        return project(self.effective_supply(name), self.height-s.last_upkeep
                       + self.p.buffer_days*self.p.blocks_per_day,
                       self.p.durable_bps, self.p)

    def expansion_price(self, name, rating):
        s = self.stations[name]
        if rating < 0:
            raise ValueError("negative rating")
        delta = max(0, tariff_numerator(rating, self.p)
                    - tariff_numerator(s.serviced_ceiling, self.p))
        duration = max(self.p.blocks_per_day, s.paid_until-self.height)
        return ceil_div(delta*duration, KW*self.p.blocks_per_day)

    def set_rating(self, name, rating, wallet):
        s = self.stations[name]
        cost = self.expansion_price(name, rating)
        if wallet < cost or (rating > s.rating and s.collateral == 0):
            raise ValueError("insufficient funds or disrepair")
        old = s.rating
        s.rating = rating
        s.serviced_ceiling = max(s.serviced_ceiling, rating)
        s.paid_capacity = s.serviced_ceiling
        s.burn += cost
        # Crucially neither health age nor paid expiry changes here.
        if rating < old:
            self.enqueue(name)
        return wallet-cost

    def pay(self, name, days, wallet, *, rating=None):
        """Atomic renewal at an optional new rating; no separate expansion fee.

        Lower-rating renewal forfeits all old larger-capacity coverage, including
        its unexpired prepaid benefit. No refunds, buckets or historical ledger.
        """
        s = self.stations[name]
        target = s.rating if rating is None else rating
        if not isinstance(days, int) or isinstance(days, bool) or days <= 0:
            raise ValueError("positive whole days required")
        until = max(self.height, s.paid_until)+days*self.p.blocks_per_day
        if until > self.height+self.p.prepay_cap_days*self.p.blocks_per_day:
            raise ValueError("prepayment cap")
        cost = upkeep_price(target, days, self.p)
        # Expanding as part of renewal must also pay for the inherited prepaid
        # time. The purchased days already fund restoration at the new rating.
        delta = max(0, tariff_numerator(target, self.p)
                    - tariff_numerator(s.serviced_ceiling, self.p))
        remaining = max(0, s.paid_until-self.height)
        cost += ceil_div(delta*remaining, KW*self.p.blocks_per_day)
        if wallet < cost or s.collateral == 0:
            raise ValueError("insufficient funds or disrepair")
        old = s.rating
        s.rating = s.serviced_ceiling = s.paid_capacity = target
        s.last_upkeep = self.height
        s.paid_until = until
        s.burn += cost
        if target < old:
            self.enqueue(name)
        return wallet-cost
