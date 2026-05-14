package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/trace"
	"strconv"
	"sync"
	"time"
)

const (
	totalTasks = 1000
	traceFile  = "trace.out"
	outputFile = "results/benchmark.csv"
)

type Workload func()

type BenchmarkResult struct {
	Workload       string
	Goroutines     int
	GOMAXPROCS     int
	TotalTimeMs    float64
	Throughput     float64
	AvgLatencyUs   float64
	MinLatencyUs   float64
	MaxLatencyUs   float64
	StdDevLatency  float64
}

func cpuWork() {
	count := 0

	for i := 2; i < 20000; i++ {
		prime := true

		for j := 2; j*j <= i; j++ {
			if i%j == 0 {
				prime = false
				break
			}
		}

		if prime {
			count++
		}
	}

	_ = count
}

func ioWork() {
	time.Sleep(2 * time.Millisecond)
}

func mixedWork() {
	cpuWork()
	time.Sleep(1 * time.Millisecond)
}

func worker(
	jobs <-chan int,
	workload Workload,
	latencies chan<- time.Duration,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for range jobs {
		start := time.Now()

		workload()

		latency := time.Since(start)
		latencies <- latency
	}
}

func runBenchmark(
	workloadName string,
	workload Workload,
	numGoroutines int,
	gomaxprocs int,
) BenchmarkResult {

	runtime.GOMAXPROCS(gomaxprocs)

	jobs := make(chan int, totalTasks)
	latencies := make(chan time.Duration, totalTasks)

	var wg sync.WaitGroup

	startTotal := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go worker(jobs, workload, latencies, &wg)
	}

	for i := 0; i < totalTasks; i++ {
		jobs <- i
	}

	close(jobs)

	wg.Wait()

	totalDuration := time.Since(startTotal)

	close(latencies)

	var latencyValues []float64

	var sum float64
	minLatency := math.MaxFloat64
	maxLatency := 0.0

	for latency := range latencies {
		us := float64(latency.Microseconds())

		latencyValues = append(latencyValues, us)

		sum += us

		if us < minLatency {
			minLatency = us
		}

		if us > maxLatency {
			maxLatency = us
		}
	}

	avgLatency := sum / float64(len(latencyValues))

	var variance float64

	for _, value := range latencyValues {
		diff := value - avgLatency
		variance += diff * diff
	}

	variance /= float64(len(latencyValues))

	stddev := math.Sqrt(variance)

	throughput := float64(totalTasks) / totalDuration.Seconds()

	return BenchmarkResult{
		Workload:      workloadName,
		Goroutines:    numGoroutines,
		GOMAXPROCS:    gomaxprocs,
		TotalTimeMs:   float64(totalDuration.Milliseconds()),
		Throughput:    throughput,
		AvgLatencyUs:  avgLatency,
		MinLatencyUs:  minLatency,
		MaxLatencyUs:  maxLatency,
		StdDevLatency: stddev,
	}
}

func writeResults(results []BenchmarkResult) error {

	err := os.MkdirAll("results", os.ModePerm)
	if err != nil {
		return err
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"workload",
		"goroutines",
		"gomaxprocs",
		"total_time_ms",
		"throughput",
		"avg_latency_us",
		"min_latency_us",
		"max_latency_us",
		"stddev_latency_us",
	}

	err = writer.Write(header)
	if err != nil {
		return err
	}

	for _, r := range results {
		record := []string{
			r.Workload,
			strconv.Itoa(r.Goroutines),
			strconv.Itoa(r.GOMAXPROCS),
			fmt.Sprintf("%.2f", r.TotalTimeMs),
			fmt.Sprintf("%.2f", r.Throughput),
			fmt.Sprintf("%.2f", r.AvgLatencyUs),
			fmt.Sprintf("%.2f", r.MinLatencyUs),
			fmt.Sprintf("%.2f", r.MaxLatencyUs),
			fmt.Sprintf("%.2f", r.StdDevLatency),
		}

		err := writer.Write(record)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {

	traceOutput, err := os.Create(traceFile)
	if err != nil {
		fmt.Println("Failed to create trace file:", err)
		return
	}
	defer traceOutput.Close()

	err = trace.Start(traceOutput)
	if err != nil {
		fmt.Println("Failed to start trace:", err)
		return
	}
	defer trace.Stop()

	workloads := map[string]Workload{
		"cpu":   cpuWork,
		"io":    ioWork,
		"mixed": mixedWork,
	}

	goroutineCounts := []int{
		1,
		2,
		4,
		8,
		16,
		32,
		64,
		128,
	}

	gomaxprocsValues := []int{
		1,
		2,
		runtime.NumCPU(),
	}

	var results []BenchmarkResult

	for workloadName, workload := range workloads {

		for _, gomaxprocs := range gomaxprocsValues {

			for _, goroutines := range goroutineCounts {

				fmt.Printf(
					"Running | workload=%s goroutines=%d GOMAXPROCS=%d\n",
					workloadName,
					goroutines,
					gomaxprocs,
				)

				result := runBenchmark(
					workloadName,
					workload,
					goroutines,
					gomaxprocs,
				)

				results = append(results, result)

				fmt.Printf(
					"Done | TotalTime=%.2f ms Throughput=%.2f ops/sec AvgLatency=%.2f us\n\n",
					result.TotalTimeMs,
					result.Throughput,
					result.AvgLatencyUs,
				)
			}
		}
	}

	err = writeResults(results)
	if err != nil {
		fmt.Println("Failed to write CSV:", err)
		return
	}

	fmt.Println("Benchmark completed successfully.")
	fmt.Println("Results saved to:", outputFile)
	fmt.Println("Trace saved to:", traceFile)
}