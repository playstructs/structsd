"""Supply-based upkeep with a lazy discount delay; research only."""
from dataclasses import dataclass

from model import Grid


@dataclass
class DiscountState:
    protected_supply: int
    reduced_since: int | None = None


class DiscountDelayGrid(Grid):
    def __init__(self, p=None, delay_days=7, restart_on_change=True):
        super().__init__(p)
        if delay_days <= 0:
            raise ValueError("positive delay required")
        self.delay_blocks = delay_days*self.p.blocks_per_day
        self.discounts = {}
        # False is an intentionally broken comparison: a shallow early reduction
        # could mature an immediate later deep reduction.
        self.restart_on_change = restart_on_change

    def add_station(self, name, root=0):
        s = super().add_station(name, root)
        self.discounts[name] = DiscountState(root)
        return s

    def quote_supply(self, name):
        supply = self.supply(name)
        d = self.discounts[name]
        if d.reduced_since is not None and self.height >= d.reduced_since+self.delay_blocks:
            return supply
        return max(supply, d.protected_supply)

    def supply_changed(self, name, previous_supply, previous_quote):
        current = self.supply(name)
        if current == previous_supply:
            return
        d = self.discounts[name]
        d.protected_supply = max(previous_quote, current)
        if current >= d.protected_supply:
            d.reduced_since = None
        elif self.restart_on_change or d.reduced_since is None:
            d.reduced_since = self.height

    def root_change(self, name, power):
        old, quote = self.supply(name), self.quote_supply(name)
        super().root_change(name, power)
        self.supply_changed(name, old, quote)

    def connect(self, source, destination, power, automated=False, accepted=True):
        old, quote = self.supply(destination), self.quote_supply(destination)
        edge = super().connect(source, destination, power, automated, accepted)
        self.supply_changed(destination, old, quote)
        return edge

    def resize(self, edge, power, admission=False):
        name = edge.destination
        old, quote = self.supply(name), self.quote_supply(name)
        super().resize(edge, power, admission)
        self.supply_changed(name, old, quote)

    def destroy(self, edge):
        name = edge.destination
        old, quote = self.supply(name), self.quote_supply(name)
        super().destroy(edge)
        self.supply_changed(name, old, quote)

    def pay(self, name, days, wallet):
        return super().pay(name, days, wallet, billable=self.quote_supply(name))
