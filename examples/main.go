package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/rhartert/srte-ls/examples/parser"
	"github.com/rhartert/srte-ls/srte"
)

var flagNetworkFile = flag.String(
	"network",
	"data/rf1239.graph",
	"",
)

var flagDemandFile = flag.String(
	"demands",
	"data/rf1239.demands",
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
	return nil
}

func main() {
	flag.Parse()
	if err := validateFlags(); err != nil {
		log.Fatalf("Error validating flags: %s", err)
	}

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

	fmt.Printf("Initial utilization: %.3f\n", state.Utilization(state.MostUtilizedEdge()))
	for iter := 0; iter < *flagMaxIterations; iter++ {
		e := state.MostUtilizedEdge()
		d := state.SelectDemand(e)

		move, found := state.Search(e, d, true)
		if !found {
			continue
		}
		state.ApplyMove(move)

		maxEdge := state.MostUtilizedEdge()
		fmt.Printf("iter %d, best utilization: %.3f\n", iter, state.Utilization(maxEdge))
	}
}
