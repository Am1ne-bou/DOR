#!/usr/bin/env python3
"""
DOR Benchmark Analyzer
Usage:
    python3 analyze_bench.py <log1> <log2> [log3] ...
    python3 analyze_bench.py --output <dir> <log1> <log2> ...

Log format expected:
    RESULT|<dest>|<status>|<retries>|<latency>ms

Status values: ACK, TIMEOUT, ABANDON
"""

import sys
import os

# ========== PARSING ==========

def parse_log(filepath):
    results = []
    with open(filepath, "r") as f:
        for line in f:
            line = line.strip()
            if not line.startswith("RESULT|"):
                continue
            parts = line.split("|")
            if len(parts) != 5:
                continue
            _, dest, status, retries, latency_str = parts
            latency = int(latency_str.replace("ms", ""))
            retries = int(retries)
            results.append({
                "dest": dest,
                "status": status,
                "retries": retries,
                "latency": latency,
            })
    return results


def compute_stats(results, label=""):
    total = len(results)
    if total == 0:
        return {
            "label": label, "total": 0,
            "ack": 0, "timeout": 0, "abandon": 0,
            "delivery_rate": 0, "avg_latency": 0,
            "median_latency": 0, "p95_latency": 0,
            "avg_retries": 0, "latencies": [],
        }

    ack_count = sum(1 for r in results if r["status"] == "ACK")
    timeout_count = sum(1 for r in results if r["status"] == "TIMEOUT")
    abandon_count = sum(1 for r in results if r["status"] == "ABANDON")

    ack_latencies = sorted([r["latency"] for r in results if r["status"] == "ACK"])
    all_retries = [r["retries"] for r in results if r["status"] == "ACK"]

    avg_latency = sum(ack_latencies) / len(ack_latencies) if ack_latencies else 0
    median_latency = ack_latencies[len(ack_latencies) // 2] if ack_latencies else 0
    p95_idx = int(len(ack_latencies) * 0.95)
    p95_latency = ack_latencies[p95_idx] if ack_latencies and p95_idx < len(ack_latencies) else 0
    avg_retries = sum(all_retries) / len(all_retries) if all_retries else 0

    return {
        "label": label, "total": total,
        "ack": ack_count, "timeout": timeout_count, "abandon": abandon_count,
        "delivery_rate": (ack_count / total) * 100,
        "avg_latency": avg_latency, "median_latency": median_latency,
        "p95_latency": p95_latency, "avg_retries": avg_retries,
        "latencies": ack_latencies,
    }


# ========== TEXT REPORT ==========

def print_report(stats_list):
    print("\n" + "=" * 60)
    print("  DOR BENCHMARK RESULTS")
    print("=" * 60)

    labels = [s["label"] for s in stats_list]
    col_w = max(20, max(len(l) for l in labels) + 2)

    header = f"{'Métrique':<25}"
    for s in stats_list:
        header += f"{s['label']:>{col_w}}"
    print(header)
    print("-" * (25 + col_w * len(stats_list)))

    rows = [
        ("Messages envoyés", "total", "d"),
        ("ACK (livrés)", "ack", "d"),
        ("Timeout", "timeout", "d"),
        ("Abandon", "abandon", "d"),
        ("Taux de livraison", "delivery_rate", ".1f", "%"),
        ("Latence moyenne", "avg_latency", ".0f", "ms"),
        ("Latence médiane", "median_latency", ".0f", "ms"),
        ("Latence P95", "p95_latency", ".0f", "ms"),
        ("Retries moyen (ACK)", "avg_retries", ".1f", ""),
    ]

    for row in rows:
        name, key, fmt = row[0], row[1], row[2]
        suffix = row[3] if len(row) > 3 else ""
        line = f"{name:<25}"
        for s in stats_list:
            val = s[key]
            line += f"{format(val, fmt) + suffix:>{col_w}}"
        print(line)

    print("=" * (25 + col_w * len(stats_list)))


# ========== GRAPHS ==========

def try_import_matplotlib():
    try:
        import matplotlib
        matplotlib.use("Agg")
        import matplotlib.pyplot as plt
        return plt
    except ImportError:
        print("\nmatplotlib non trouvé. Installe avec:")
        print("  pip install matplotlib --break-system-packages")
        sys.exit(1)


def generate_graphs(stats_list, output_dir="."):
    plt = try_import_matplotlib()

    labels = [s["label"] for s in stats_list]
    colors = ["#e74c3c", "#2ecc71", "#3498db", "#f39c12", "#9b59b6"]

    # Graph 1: Taux de livraison
    fig, ax = plt.subplots(figsize=(8, 5))
    rates = [s["delivery_rate"] for s in stats_list]
    bars = ax.bar(labels, rates, color=colors[:len(labels)], edgecolor="white", linewidth=1.5)
    for bar, rate in zip(bars, rates):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 1,
                f"{rate:.1f}%", ha="center", va="bottom", fontweight="bold", fontsize=12)
    ax.set_ylim(0, 110)
    ax.set_ylabel("Taux de livraison (%)", fontsize=12)
    ax.set_title("Taux de livraison des messages", fontsize=14, fontweight="bold")
    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    plt.tight_layout()
    path1 = os.path.join(output_dir, "01_delivery_rate.png")
    plt.savefig(path1, dpi=150)
    plt.close()
    print(f"  -> {path1}")

    # Graph 2: Répartition des résultats
    fig, ax = plt.subplots(figsize=(8, 5))
    ack_pcts = [s["ack"] / s["total"] * 100 if s["total"] > 0 else 0 for s in stats_list]
    timeout_pcts = [s["timeout"] / s["total"] * 100 if s["total"] > 0 else 0 for s in stats_list]
    abandon_pcts = [s["abandon"] / s["total"] * 100 if s["total"] > 0 else 0 for s in stats_list]
    x = range(len(labels))
    ax.bar(x, ack_pcts, label="ACK (livré)", color="#2ecc71")
    ax.bar(x, timeout_pcts, bottom=ack_pcts, label="Timeout", color="#e74c3c")
    bottom2 = [a + t for a, t in zip(ack_pcts, timeout_pcts)]
    ax.bar(x, abandon_pcts, bottom=bottom2, label="Abandon", color="#95a5a6")
    ax.set_xticks(list(x))
    ax.set_xticklabels(labels)
    ax.set_ylim(0, 105)
    ax.set_ylabel("Pourcentage (%)", fontsize=12)
    ax.set_title("Répartition des résultats", fontsize=14, fontweight="bold")
    ax.legend(loc="upper right")
    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    plt.tight_layout()
    path2 = os.path.join(output_dir, "02_results_breakdown.png")
    plt.savefig(path2, dpi=150)
    plt.close()
    print(f"  -> {path2}")

    # Graph 3: Latences (box plot)
    fig, ax = plt.subplots(figsize=(8, 5))
    latency_data = [s["latencies"] for s in stats_list]
    has_data = any(len(d) > 0 for d in latency_data)
    if has_data:
        bp = ax.boxplot(latency_data, tick_labels=labels, patch_artist=True)
        for patch, color in zip(bp["boxes"], colors):
            patch.set_facecolor(color)
            patch.set_alpha(0.6)
        ax.set_ylabel("Latence (ms)", fontsize=12)
        ax.set_title("Distribution des latences (messages livrés)", fontsize=14, fontweight="bold")
        ax.spines["top"].set_visible(False)
        ax.spines["right"].set_visible(False)
    else:
        ax.text(0.5, 0.5, "Pas de données de latence", transform=ax.transAxes,
                ha="center", va="center", fontsize=14)
    plt.tight_layout()
    path3 = os.path.join(output_dir, "03_latency_boxplot.png")
    plt.savefig(path3, dpi=150)
    plt.close()
    print(f"  -> {path3}")

    # Graph 4: Latence moyenne + P95
    fig, ax = plt.subplots(figsize=(8, 5))
    x = range(len(labels))
    width = 0.35
    avg_lats = [s["avg_latency"] for s in stats_list]
    p95_lats = [s["p95_latency"] for s in stats_list]
    ax.bar([i - width/2 for i in x], avg_lats, width, label="Moyenne", color="#3498db")
    ax.bar([i + width/2 for i in x], p95_lats, width, label="P95", color="#e74c3c")
    ax.set_xticks(list(x))
    ax.set_xticklabels(labels)
    ax.set_ylabel("Latence (ms)", fontsize=12)
    ax.set_title("Latence moyenne vs P95", fontsize=14, fontweight="bold")
    ax.legend()
    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    plt.tight_layout()
    path4 = os.path.join(output_dir, "04_latency_comparison.png")
    plt.savefig(path4, dpi=150)
    plt.close()
    print(f"  -> {path4}")

    # Graph 5: Retries moyen
    fig, ax = plt.subplots(figsize=(8, 5))
    avg_retries = [s["avg_retries"] for s in stats_list]
    bars = ax.bar(labels, avg_retries, color=colors[:len(labels)], edgecolor="white", linewidth=1.5)
    for bar, val in zip(bars, avg_retries):
        if val > 0:
            ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 0.05,
                    f"{val:.1f}", ha="center", va="bottom", fontweight="bold", fontsize=12)
    ax.set_ylabel("Nombre moyen de retries", fontsize=12)
    ax.set_title("Retries moyen par message livré", fontsize=14, fontweight="bold")
    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    plt.tight_layout()
    path5 = os.path.join(output_dir, "05_avg_retries.png")
    plt.savefig(path5, dpi=150)
    plt.close()
    print(f"  -> {path5}")

    return [path1, path2, path3, path4, path5]


# ========== MAIN ==========

def main():
    output_dir = "dor_graphs"
    files = []

    args = sys.argv[1:]
    i = 0
    while i < len(args):
        if args[i] == "--output" and i + 1 < len(args):
            output_dir = args[i + 1]
            i += 2
        else:
            files.append(args[i])
            i += 1

    if len(files) == 0:
        print("Usage:")
        print("  python3 analyze_bench.py <log1> <log2> [log3] ...")
        print("  python3 analyze_bench.py --output <dir> <log1> <log2> ...")
        sys.exit(1)

    all_stats = []
    for filepath in files:
        if not os.path.exists(filepath):
            print(f"ERREUR: {filepath} n'existe pas")
            continue
        label = os.path.basename(filepath).replace(".log", "").replace("_", " ")
        results = parse_log(filepath)
        if len(results) == 0:
            print(f"ATTENTION: Aucune ligne RESULT trouvée dans {filepath}")
            continue
        stats = compute_stats(results, label)
        all_stats.append(stats)

    if len(all_stats) == 0:
        print("Aucun résultat à analyser.")
        sys.exit(1)

    print_report(all_stats)

    # Sauvegarder le rapport texte
    report_path = os.path.join(output_dir, "report.txt")
    with open(report_path, "w") as f:
        import io
        old_stdout = sys.stdout
        sys.stdout = io.StringIO()
        print_report(all_stats)
        report_text = sys.stdout.getvalue()
        sys.stdout = old_stdout
        f.write(report_text)
    print(f"Rapport sauvegardé dans {report_path}")

    os.makedirs(output_dir, exist_ok=True)
    print(f"\nGénération des graphiques dans {output_dir}/...")
    graphs = generate_graphs(all_stats, output_dir)
    print(f"\nTerminé ! {len(graphs)} graphiques générés dans {output_dir}/")


if __name__ == "__main__":
    main()