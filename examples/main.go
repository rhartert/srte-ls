package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"

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
	1000,
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

	state := srteState()
	rng := rand.New(rand.NewSource(*flagSeed))
	nEdges := len(state.Instance.Graph.Edges)

	// Datastructure to enable fast random selection of edges.
	edgeWheel := srte.NewSumTree(nEdges)
	for e := 0; e < nEdges; e++ {
		edgeWheel.SetWeight(e, math.Pow(state.Utilization(e), *flagAlpha))
	}

	maxUtil := state.Utilization(state.MostUtilizedEdge())
	fmt.Printf("Initial utilization: %.3f\n", maxUtil)
	for iter := 0; iter < *flagMaxIterations; iter++ {
		// Randomly select the next edge to improve. The most utilized an edge
		// is, the most likely it is to be selected.
		e := edgeWheel.Get(rng.Float64() * edgeWheel.TotalWeight())

		// Select the demand with the highest traffic on the edge.
		d := state.SelectDemand(e)

		// Search for a move that reduces the load of the selected edge and does
		// not increase the maximum utilization of the network.
		move, found := state.Search(e, d, maxUtil)
		if !found {
			continue
		}

		// Apply the move but do not persist the changes yet (see below).
		state.ApplyMove(move, false)

		// Update structures for fast selection by iterating on the edges that
		// were impacted by the move.
		for _, lc := range state.Changes() {
			edgeWheel.SetWeight(lc.Edge, math.Pow(state.Utilization(lc.Edge), *flagAlpha))
		}

		// Persist the changes now that the structures have been updated.
		state.PersistChanges()

		maxUtil = state.Utilization(state.MostUtilizedEdge())
		fmt.Printf("iter %d, best utilization: %f\n", iter, maxUtil)
	}
}
