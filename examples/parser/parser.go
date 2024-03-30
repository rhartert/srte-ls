package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rhartert/srte-ls/srte"
)

func ParseDemands(filepath string) ([]srte.Demand, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	demands := []srte.Demand{}
	for i := 0; scanner.Scan(); i++ {
		if i <= 1 {
			continue
		}
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid demand: missing parts")
		}
		from, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid demand: %s", err)
		}
		to, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid demand: %s", err)
		}
		bw, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, fmt.Errorf("invalid demand: %s", err)
		}
		demands = append(demands, srte.Demand{
			From:      from,
			To:        to,
			Bandwidth: int64(bw),
		})
	}

	return demands, nil
}

func ParseNetwork(filepath string) (*srte.Digraph, []int64, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Scan()
	parts := strings.Split(scanner.Text(), " ")
	nNodes, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, nil, err
	}

	scanner.Scan() // skip headers
	for i := 0; i < nNodes; i++ {
		scanner.Scan() // skip all node labels
	}
	scanner.Scan() // skip line between node and edge sections
	scanner.Scan() // skip edge headers
	scanner.Scan() // skip number of edges

	edges := []srte.Edge{}
	capacities := []int64{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) != 6 {
			return nil, nil, fmt.Errorf("invalid demand: missing parts")
		}
		from, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid demand: %s", err)
		}
		to, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid demand: %s", err)
		}
		w, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid demand: %s", err)
		}
		bw, err := strconv.Atoi(parts[4])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid demand: %s", err)
		}

		capacities = append(capacities, int64(bw))
		edges = append(edges, srte.Edge{
			From: from,
			To:   to,
			Cost: w,
		})
	}

	return srte.NewDigraph(edges, nNodes), capacities, nil
}
