package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"

	"github.com/rhartert/srte-ls/examples/parser"
	"github.com/rhartert/srte-ls/srte"
	"github.com/rhartert/yagh"
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

func selectDemand(demands map[int]int64) int {
	bestLoad := int64(0)
	bestDemand := -1
	for d, l := range demands {
		switch {
		case l == bestLoad && d < bestDemand:
			bestDemand = d
		case bestLoad < l:
			bestLoad = l
			bestDemand = d
		}
	}
	return bestDemand
}

func main() {
	flag.Parse()
	if err := validateFlags(); err != nil {
		log.Fatalf("Error validating flags: %s", err)
	}

	state := srteState()
	rng := rand.New(rand.NewSource(*flagSeed))
	nEdges := len(state.Instance.Graph.Edges)

	// Enable fast random selection of edges.
	edgeWheel := srte.NewSumTree(nEdges)
	// Maintain the most utilized edge.
	edgesByUtilization := yagh.New[float64](nEdges)
	// Set of demands (and traffic) for each edge.
	edgesToDemand := make([]map[int]int64, nEdges)

	for e := 0; e < nEdges; e++ {
		edgeWheel.SetWeight(e, math.Pow(state.Utilization(e), *flagAlpha))
		edgesByUtilization.Put(e, -state.Utilization(e)) // non-decreasing order
		edgesToDemand[e] = map[int]int64{}
	}

	for i, d := range state.Instance.Demands {
		for _, er := range state.FGraphs.EdgeRatios(d.From, d.To) {
			edgesToDemand[er.Edge][i] = srte.SplitLoad(d.Bandwidth, er.Ratio)
		}
	}

	mostUtilizedEdge, _ := edgesByUtilization.Min()
	maxUtil := state.Utilization(mostUtilizedEdge.Elem)
	fmt.Printf("Initial utilization: %.3f\n", maxUtil)
	for iter := 0; iter < *flagMaxIterations; iter++ {
		// Randomly select the next edge to improve. The most utilized an edge
		// is, the most likely it is to be selected.
		e := edgeWheel.Get(rng.Float64() * edgeWheel.TotalWeight())

		// Select the demand with the highest traffic on the edge.
		d := selectDemand(edgesToDemand[e])

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
			util := state.Utilization(lc.Edge)
			edgeWheel.SetWeight(lc.Edge, math.Pow(util, *flagAlpha))
			edgesByUtilization.Put(lc.Edge, -util) // non-decreasing order

			// Efficiently maintain the list of demands on each edge by
			// comparing the load change and how much traffic the demand was
			// sending on the edge prior to the change.
			prev := edgesToDemand[lc.Edge][d]
			switch delta := state.Load(lc.Edge) - lc.PreviousLoad; {
			case prev == 0: // the demand was not sending traffic before
				edgesToDemand[lc.Edge][d] = delta
			case prev == -delta: // all the demand's traffic is removed
				delete(edgesToDemand[lc.Edge], d)
			default: // the demand traffic has changed but is non-null
				edgesToDemand[lc.Edge][d] += delta
			}
		}

		// Persist the changes now that the structures have been updated.
		state.PersistChanges()

		mostUtilizedEdge, _ = edgesByUtilization.Min()
		maxUtil = state.Utilization(mostUtilizedEdge.Elem)
		fmt.Printf("iter %d, best utilization: %f\n", iter, maxUtil)
	}
}
