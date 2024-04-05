package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/rhartert/srte-ls/examples/parser"
	"github.com/rhartert/srte-ls/solver"
	"github.com/rhartert/srte-ls/srte"
)

var flagNetworkFile = flag.String(
	"network",
	"data/synth100.graph",
	"",
)

var flagDemandFile = flag.String(
	"demands",
	"data/synth100.demands",
	"",
)

var flagUseUnaryWeights = flag.Bool(
	"unary_weights",
	true,
	"",
)

var flagScaling = flag.Int64(
	"scaling",
	1000,
	"",
)

var flagMaxNodesPerPath = flag.Int(
	"max_nodes",
	4,
	"",
)

var flagMaxIterations = flag.Int(
	"max_iterations",
	10000,
	"",
)

var flagSeed = flag.Int64(
	"seed",
	42,
	"",
)

var flagAlpha = flag.Float64(
	"alpha",
	8.0,
	"",
)

var flagBeta = flag.Float64(
	"beta",
	2.0,
	"",
)

func validateFlags() error {
	if *flagNetworkFile == "" {
		return fmt.Errorf("missing network file")
	}
	if *flagDemandFile == "" {
		return fmt.Errorf("missing demands file")
	}
	if n := *flagScaling; n <= 0 {
		return fmt.Errorf("scaling should be greater than 0, got %d", n)
	}
	if n := *flagMaxNodesPerPath; n <= 0 {
		return fmt.Errorf("paths must have at least 1 intermediate nodes, got: %d", n)
	}
	if n := *flagMaxIterations; n < 0 {
		return fmt.Errorf("number of iterations should be positive, got: %d", n)
	}
	if n := *flagAlpha; n < 0 {
		return fmt.Errorf("parameter alpha must be non-negative, got: %f", n)
	}
	if n := *flagBeta; n < 0 {
		return fmt.Errorf("parameter beta must be non-negative, got: %f", n)
	}
	return nil
}

func srteState() *srte.SRTE {
	network, capacities, err := parser.ParseNetwork(*flagNetworkFile)
	if err != nil {
		log.Fatalf("Error reading graph file: %s", err)
	}
	demands, err := parser.ParseDemands(*flagDemandFile)
	if err != nil {
		log.Fatalf("Error reading demand file: %s", err)
	}

	if s := *flagScaling; s > 1 {
		for i := range demands {
			demands[i].Bandwidth *= s
		}
		for i := range capacities {
			capacities[i] *= s
		}
	}
	if *flagUseUnaryWeights {
		for i := range network.Edges {
			network.Edges[i].Cost = 1
		}
	}

	fgs, err := srte.NewFGraphs(network)
	if err != nil {
		log.Fatal(err)
	}

	state, err := srte.NewSRTE(&srte.SRTEInstance{
		Graph:          network,
		FGraphs:        fgs,
		MaxPathNodes:   *flagMaxNodesPerPath + 2, // + source and destination
		Demands:        demands,
		LinkCapacities: capacities,
	})
	if err != nil {
		log.Fatal(err)
	}

	return state
}

func main() {
	flag.Parse()
	if err := validateFlags(); err != nil {
		log.Fatalf("Error validating flags: %s", err)
	}

	parseStart := time.Now()

	rng := rand.New(rand.NewSource(*flagSeed))
	lgs := solver.NewLinkGuidedSolver(srteState(), solver.Config{
		Alpha: *flagAlpha,
		Beta:  *flagBeta,
	})

	startUtil := lgs.MaxUtilization()
	optStart := time.Now()

	for iter := 0; iter < *flagMaxIterations; iter++ {
		// Randomly select the next edge to improve. The more utilized an edge
		// is, the more likely it is to be selected.
		e := lgs.SelectEdge(rng.Float64())

		// Select the demand to move. The more a demand contributes to the
		// edge's load, the more likely it is to be selected.
		d := lgs.SelectDemand(e, rng.Float64())

		// Search for a move that reduces the load of the selected edge and does
		// not increase the maximum utilization of the network.
		move, found := lgs.Search(e, d, lgs.MaxUtilization())
		if !found {
			continue
		}
		lgs.ApplyMove(move)
	}

	totalTime := time.Since(parseStart)
	optTime := time.Since(optStart)
	fmt.Printf("total time (ms):        %v\n", totalTime.Milliseconds())
	fmt.Printf("optimization time (ms): %v\n", optTime.Milliseconds())
	fmt.Printf("utilization (before):   %f\n", startUtil)
	fmt.Printf("utilization (after):    %f\n", lgs.MaxUtilization())
}
