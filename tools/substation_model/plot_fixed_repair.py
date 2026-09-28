"""Optional static figures for the fixed repair experiment; needs matplotlib."""
import argparse
import json

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from pathlib import Path


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("results", type=Path)
    args = parser.parse_args()
    r = json.loads(args.results.read_text())
    plt.rcParams.update({"font.family": "DejaVu Sans", "font.size": 10,
                         "axes.spines.top": False, "axes.spines.right": False})
    fig, axs = plt.subplots(2, 2, figsize=(13, 9), layout="constrained")
    colors = ["#176B91", "#C35818", "#7246A0"]
    ax = axs[0, 0]
    rows = [x for x in r["cadence"] if x["baseline_bps"] == 7000
            and x["repair_alpha"] == 10 and x["service_possible"]]
    ax.plot([x["supply_kw"]/x["demand_kw"] for x in rows],
            [x["latest_repair_days"] for x in rows], "o-", color=colors[0])
    ax.axhline(30, color="#666666", linestyle=":", label="Health reaches zero")
    ax.set(xscale="log", xlabel="Raw supply / demand (log scale)", ylabel="Days between repairs",
           title="A. More spare supply permits longer repair intervals")
    ax.legend(fontsize=8)
    ax = axs[0, 1]
    rows = [x for x in r["partition"] if x["repair_alpha"] == 10]
    ax.plot([x["parallel_stations"] for x in rows], [x["network_alpha_per_day"] for x in rows],
            "o-", color=colors[1])
    ax.set(xscale="log", yscale="log", xlabel="Parallel stations (log scale)",
           ylabel="Network repair burn: Alpha/day (log scale)",
           title="B. Same total supply and demand: more stations cost more")
    ax = axs[1, 0]
    fees = r["alternative"]["repair_prices_alpha"]
    rates = [1, 10, 25, 50, 100]
    cells = [[next(x["full_online_fraction"]*100 for x in r["treasury"]
                   if x["repair_alpha"] == fee and x["treasury_income_bps"] == rate
                   and not x["one_repair_prefunded"] and not x["essential_fallback"])
              for rate in rates] for fee in fees]
    ax.imshow(cells, cmap="Blues", vmin=0, vmax=100, aspect="auto")
    for y, row in enumerate(cells):
        for x, value in enumerate(row):
            ax.text(x, y, f"{value:.0f}%", ha="center", va="center",
                    color="white" if value > 60 else "#222222")
    ax.set(xticks=range(len(rates)), xticklabels=[f"{v/100:g}%" for v in rates],
           yticks=range(len(fees)), yticklabels=fees, xlabel="Online income reaching treasury",
           ylabel="Fixed repair fee (Alpha)", title="C. Service uptime over 90 days; treasury starts empty")
    ax = axs[1, 1]
    for rate, color in zip([0, 1000, 5000], colors):
        rows = [x for x in r["glut"] if x["repair_alpha"] == 10 and x["investment_bps"] == rate]
        ax.plot([x["day"] for x in rows], [x["available_demand_ratio"] for x in rows],
                label=f"{rate/100:g}% issuance reinvested", color=color)
    ax.axhline(1, color="#444444", linestyle=":")
    ax.set(xlabel="Day", ylabel="Available capacity / fixed demand",
           title="D. Flat repairs do not prevent a permanent energy glut")
    ax.legend(fontsize=8)
    for i in [0, 1, 3]:
        axs.flat[i].grid(alpha=.15)
    fig.suptitle("Fixed-price substation repair — simpler accounting, strong consolidation incentive\n"
                 "Candidate rules: 70% baseline, 30-day wear, no temporary bonus or scheduled enforcement",
                 fontsize=14)
    for ext in ["png", "svg"]:
        fig.savefig(args.results.parent/f"fixed-repair-scenarios.{ext}", dpi=160)


if __name__ == "__main__":
    main()
