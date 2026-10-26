"""Pure arithmetic candidate for charging service added after an upkeep purchase.

No bank calls, chain hooks or automatic-allocation policy are implemented here.
"""
from dataclasses import dataclass

from model import ALPHA, KW, Parameters, ceil_div
from rating_model import tariff_numerator


@dataclass
class ServiceCoverage:
    p: Parameters = Parameters()
    base_fee: int = ALPHA
    player_fee: int = ALPHA//20
    covered_players: int = 0
    covered_power: int = 0
    repaired_at: int = 0
    paid_until: int = 0
    burned: int = 0

    def delta_numerator(self, players, power):
        if players < 0 or power < 0:
            raise ValueError('negative service')
        return (max(0, players-self.covered_players)*self.player_fee*KW
                + max(0, tariff_numerator(power, self.p)-tariff_numerator(self.covered_power, self.p)))

    def add_service(self, height, players, power, wallet):
        duration = max(self.p.blocks_per_day, self.paid_until-height)
        cost = ceil_div(self.delta_numerator(players, power)*duration,
                        KW*self.p.blocks_per_day)
        if wallet < cost:
            raise ValueError('insufficient funds')
        self.covered_players = max(self.covered_players, players)
        self.covered_power = max(self.covered_power, power)
        self.burned += cost
        # Existing health age is inherited. New service never repairs old service.
        return wallet-cost

    def repair(self, height, days, players, power, wallet):
        if not isinstance(days, int) or days <= 0 or players < 0 or power < 0:
            raise ValueError('invalid repair')
        until = max(height, self.paid_until)+days*self.p.blocks_per_day
        if until > height+self.p.prepay_cap_days*self.p.blocks_per_day:
            raise ValueError('prepayment cap')
        daily = (self.base_fee+players*self.player_fee)*KW+tariff_numerator(power, self.p)
        cost = ceil_div(daily*days, KW)
        cost += ceil_div(self.delta_numerator(players, power)*max(0, self.paid_until-height),
                         KW*self.p.blocks_per_day)
        if wallet < cost:
            raise ValueError('insufficient funds')
        self.covered_players, self.covered_power = players, power
        self.paid_until, self.repaired_at = until, height
        self.burned += cost
        return wallet-cost
