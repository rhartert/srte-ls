> 🫵 **Looking for collaborators** who would be interested in integrating this 
optimizer (or [a more powerful version](https://scholar.google.com/citations?view_op=view_citation&hl=en&user=1jocmcIAAAAJ&citation_for_view=1jocmcIAAAAJ:Wp0gIr-vW9MCs)) into a real-world network 
controller. If you're such a person, please contact me at ren.hartert@gmail.com. 

# SRTE-LS ⚡️

SRTE-LS is a fast optimizer to find (near) optimal traffic placement in Segment Routing enabled networks. It is a Go implementation of the Link-Guided Local Search algorithm presented in [[1](https://research.google/pubs/expect-the-unexpected-sub-second-optimization-for-segment-routing/), [2](https://scholar.google.com/citations?view_op=view_citation&hl=en&user=1jocmcIAAAAJ&citation_for_view=1jocmcIAAAAJ:_Qo2XoVZTnwC)].

## Project Description

Segment Routing is a network technology that allows network operators to steer traffic along a specific path through the network by encoding instructions in the packet headers. Optimizing traffic placement in such networks is crucial for improving network performance, reducing congestion, and enhancing overall efficiency. SRTE-LS aims to address this optimization problem by efficiently determining the best placement of traffic flows within a Segment Routing network.

## How to Use

### Installation

Before using SRTE-LS, ensure you have Go installed on your system. You can install it from the [official Go website](https://golang.org/).

Clone the repository:

```sh
git clone https://github.com/rhartert/srte-ls.git
cd srte-ls
```

### Usage

The `examples` directory contains a complete implementation of a LGS solver. Follow these steps to run it:

```sh
cd examples
go build
./examples --max_iterations=10000
```

Upon successful execution, you should see output similar to the following:

```
total time (ms):        314
optimization time (ms): 236
utilization (before):   2.325262
utilization (after):    0.915767
```

## License

This project is licensed under the [MIT License](LICENSE).