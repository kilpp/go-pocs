package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
)

// fibonacci recursively calculates the nth Fibonacci number.
// This is an intentionally inefficient implementation to demonstrate CPU profiling.
func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

// bubbleSort sorts a slice of integers using the bubble sort algorithm.
// This is an intentionally inefficient implementation to demonstrate CPU profiling.
func bubbleSort(arr []int) {
	n := len(arr)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if arr[j] > arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
			}
		}
	}
}

// memoryHog allocates a significant amount of memory.
// This is to demonstrate memory profiling.
func memoryHog() {
	// Allocate a large slice of slices of integers
	hog := make([][]int, 1000)
	for i := range hog {
		hog[i] = make([]int, 10000)
	}
}

func main() {
	// --- CPU Profiling ---
	cpuFile, err := os.Create("cpu.pprof")
	if err != nil {
		fmt.Println("could not create CPU profile: ", err)
		return
	}
	defer cpuFile.Close()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		fmt.Println("could not start CPU profile: ", err)
		return
	}
	defer pprof.StopCPUProfile()

	// Run some CPU-intensive tasks
	fmt.Println("Fibonacci(35) =", fibonacci(35))

	arr := make([]int, 1000)
	for i := 0; i < len(arr); i++ {
		arr[i] = rand.Intn(len(arr))
	}
	bubbleSort(arr)
	fmt.Println("Sorted a 1000-element array.")

	// --- Memory Profiling ---
	memFile, err := os.Create("mem.pprof")
	if err != nil {
		fmt.Println("could not create memory profile: ", err)
		return
	}
	defer memFile.Close()

	// Run a memory-intensive task
	memoryHog()

	// Write the memory profile
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		fmt.Println("could not write memory profile: ", err)
		return
	}

	fmt.Println("Profiling data written to cpu.pprof and mem.pprof")
	fmt.Println("Use 'go tool pprof cpu.pprof' and 'go tool pprof mem.pprof' to analyze.")
}
