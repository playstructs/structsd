"""Optional research figure. Requires matplotlib; the model does not."""
import argparse
import json
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt


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
    for bps, color in zip([0, 1000, 5000], colors):
        rows = [x for x in r["glut"] if x["generator_kw_per_alpha"] == 2
                and x["investment_bps_of_issuance"] == bps]
        ax.plot([x["day"] for x in rows], [x["supply_demand_ratio"] for x in rows],
                label=f"{bps/100:g}% issuance reinvested", color=color, linewidth=1.8)
    ax.axhline(1, color="#444444", linestyle=":", label="Demand met")
    ax.set(title="A. Permanent generation can deepen the glut", xlabel="Simulation day",
           ylabel="Available capacity / fixed demand")
    ax.legend(fontsize=8)
    ax = axs[0, 1]
    for fragments, color in zip([1, 100], colors):
        rows = [x for x in r["oldest_first"] if x["oldest_pct"] == 99 and x["old_fragments"] == fragments]
        ax.plot([x["withdrawal_pct"] for x in rows], [x["commitments_lost_pct"] for x in rows],
                "o-", label=f"Older group uses {fragments} allocation(s)", color=color)
    ax.set(title="B. Small cuts can destroy large oldest commitments", xlabel="Accepted input withdrawn (%)",
           ylabel="Outgoing commitments destroyed (%)", xlim=(0, 22), ylim=(-3, 103))
    ax.legend(fontsize=8)
    ax = axs[1, 0]
    xs = r["ghost_power"]
    ax.plot([x["day"] for x in xs], [x["relay_throughput_kw"]/1000 for x in xs],
            label="Neglected relay: projected output", color=colors[1], linewidth=2)
    ax.plot([x["day"] for x in xs], [x["leaf_available_kw"]/1000 for x in xs],
            label="Maintained downstream station: available", color=colors[0], linewidth=1.8)
    ax.axvline(30, color="#777777", linestyle=":")
    ax.scatter([60], [0], marker="X", s=60, color="#222222", label="Valid audit at day 60 cuts downstream to zero")
    ax.set(title="C. Enforcement depends on somebody auditing", xlabel="Day without relay upkeep",
           ylabel="Capacity (MW)")
    ax.legend(fontsize=8, loc="lower left")
    ax = axs[1, 1]
    for mode, label, color in [("fixed_load", "Keep full load", colors[1]),
                              ("shed_to_essential", "Shed to essential load", colors[0])]:
        rows = [x for x in r["treasury"] if x["initial_alpha"] == 0 and x["load_policy"] == mode]
        ax.plot([x["treasury_income_bps"]/100 for x in rows],
                [x["earning_fraction"]*100 for x in rows], "o-", label=label, color=color)
    ax.set(title="D. Treasury funding and fallback load matter", xlabel="Online income contributed to treasury (%)",
           ylabel="Time able to earn over 90 days (%)", xlim=(-.03,1.05), ylim=(0,105))
    ax.legend(fontsize=8)
    for ax in axs.flat:
        ax.grid(alpha=.15)
    fig.suptitle("Substation adversarial model — candidate rules, not live-chain outcomes\n"
                 "No automatic upkeep enforcement; integer health/pricing; scenario assumptions apply", fontsize=15)
    for ext in ["png", "svg"]:
        fig.savefig(args.results.parent/f"collapse-scenarios.{ext}", dpi=160)


if __name__ == "__main__":
    main()
