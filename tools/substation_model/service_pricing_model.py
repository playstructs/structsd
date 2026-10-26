"""Upkeep priced by connected players and outgoing commitments. Research only."""
from model import ALPHA, Grid, upkeep_price


class ServicePricingGrid(Grid):
    def __init__(self, p=None, base_fee=ALPHA, player_fee=ALPHA//20):
        super().__init__(p)
        if base_fee < 0 or player_fee < 0:
            raise ValueError("negative price")
        self.base_fee = base_fee
        self.player_fee = player_fee
        self.connections = {}
        self.connection_counts = {}
        self.edge_types = {}

    def add_station(self, name, root=0):
        s = super().add_station(name, root)
        self.connection_counts[name] = 0
        return s

    def move_player(self, player, destination, authorized=True):
        # Abstracts the existing permission-gated connect/migrate/disconnect paths.
        # It is not an implementation of the actual permission resolver or gas.
        if not authorized or (destination is not None and destination not in self.stations):
            raise ValueError("unauthorized or missing station")
        previous = self.connections.get(player)
        if previous == destination:
            return
        if previous is not None:
            self.connection_counts[previous] -= 1
        if destination is None:
            self.connections.pop(player, None)
        else:
            self.connections[player] = destination
            self.connection_counts[destination] += 1

    def player_capacity(self, player):
        station = self.connections.get(player)
        if station is None:
            return 0
        return self.shared(station)//self.connection_counts[station]

    def quote(self, name, days=1):
        if not isinstance(days, int) or isinstance(days, bool) or days <= 0:
            raise ValueError("positive whole days required")
        power = self.load(name)
        outgoing = upkeep_price(power, days, self.p) if power else 0
        return (self.base_fee+self.player_fee*self.connection_counts[name])*days+outgoing

    def pay(self, name, days, wallet):
        s = self.stations[name]
        cost = self.quote(name, days)
        until = max(self.height, s.paid_until)+days*self.p.blocks_per_day
        if until > self.height+self.p.prepay_cap_days*self.p.blocks_per_day:
            raise ValueError("prepayment cap")
        if wallet < cost or s.collateral == 0:
            raise ValueError("insufficient funds or disrepair")
        s.burn += cost
        s.last_upkeep = self.height
        s.paid_until = until
        # Conservative bonus control from the original kernel. Health still
        # resets for the whole station, independently of later customer changes.
        s.paid_capacity = self.supply(name)
        return wallet-cost

    def connect(self, source, destination, power, automated=False, accepted=True,
                kind="dynamic"):
        if kind not in ("dynamic", "static", "provider", "automated"):
            raise ValueError("unknown allocation kind")
        e = super().connect(source, destination, power, automated or kind == "automated", accepted)
        self.edge_types[e.index] = "automated" if e.automated else kind
        return e

    def resize_dynamic(self, edge, power, authorized=True):
        if not authorized or self.edge_types[edge.index] != "dynamic" or power <= 0:
            raise ValueError("not an authorized positive dynamic update")
        # Conservatively enforce the proposed durable admission budget on growth.
        super().resize(edge, power, admission=power > edge.power)
