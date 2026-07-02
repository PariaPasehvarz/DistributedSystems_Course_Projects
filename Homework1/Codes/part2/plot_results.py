import pandas as pd
import matplotlib.pyplot as plt

df = pd.read_csv("results/benchmark.csv")


for workload in df["workload"].unique():

    subset = df[df["workload"] == workload]

    plt.figure(figsize=(8, 5))

    for procs in subset["gomaxprocs"].unique():

        data = subset[subset["gomaxprocs"] == procs]

        plt.plot(
            data["goroutines"],
            data["throughput"],
            marker='o',
            label=f"GOMAXPROCS={procs}"
        )

    plt.title(f"Throughput vs Goroutines ({workload})")
    plt.xlabel("Number of Goroutines")
    plt.ylabel("Throughput (ops/sec)")
    plt.xscale("log", base=2)
    plt.grid(True)
    plt.legend()

    plt.savefig(f"results/throughput_{workload}.png")
    plt.close()

for workload in df["workload"].unique():

    subset = df[df["workload"] == workload]

    plt.figure(figsize=(8, 5))

    for procs in subset["gomaxprocs"].unique():

        data = subset[subset["gomaxprocs"] == procs]

        plt.plot(
            data["goroutines"],
            data["total_time_ms"],
            marker='o',
            label=f"GOMAXPROCS={procs}"
        )

    plt.title(f"Total Time vs Goroutines ({workload})")
    plt.xlabel("Number of Goroutines")
    plt.ylabel("Total Time (ms)")
    plt.xscale("log", base=2)
    plt.grid(True)
    plt.legend()

    plt.savefig(f"results/time_{workload}.png")
    plt.close()


for workload in df["workload"].unique():

    subset = df[df["workload"] == workload]

    plt.figure(figsize=(8, 5))

    for procs in subset["gomaxprocs"].unique():

        data = subset[subset["gomaxprocs"] == procs]

        plt.plot(
            data["goroutines"],
            data["avg_latency_us"],
            marker='o',
            label=f"GOMAXPROCS={procs}"
        )

    plt.title(f"Average Latency vs Goroutines ({workload})")
    plt.xlabel("Number of Goroutines")
    plt.ylabel("Average Latency (µs)")
    plt.xscale("log", base=2)
    plt.grid(True)
    plt.legend()

    plt.savefig(f"results/latency_{workload}.png")
    plt.close()

print("Plots generated successfully.")