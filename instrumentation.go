package ilp

import (
	"fmt"
	"io"
)

type bnbMiddleware interface {

	// Receives each subproblem solution and a corresponding decision
	ProcessDecision(solution, bnbDecision)
}

type dummyMiddleware struct{}

func (d dummyMiddleware) ProcessDecision(s solution, b bnbDecision) {
	return
}

type treeLogger struct {
	nodes []node
}

// represents a node from the enumeration tree.
type node struct {
	id     int64
	parent int64

	// objective function value
	z float64

	// intermediate solution
	x []float64

	decision bnbDecision
}

// convert a solution to a node, assiging an integer id.
func newNode(soln solution, d bnbDecision) node {
	return node{
		id:       soln.problem.id,
		parent:   soln.problem.parent,
		z:        soln.z,
		x:        soln.x,
		decision: d,
	}
}

func (t *treeLogger) ProcessDecision(s solution, d bnbDecision) {
	t.nodes = append(t.nodes, newNode(s, d))
}

// takes an io.Writer to write the DOT-file visualisation of the processed enumeration tree to.
func (t *treeLogger) toDOT(out io.Writer) {

	writeRow := func(r string, args ...interface{}) {
		if len(args) > 0 {
			out.Write([]byte(fmt.Sprintf(r, args...)))
		} else {
			out.Write([]byte(r))
		}

		out.Write([]byte("\n"))
	}

	// write DOT-file start boilerplate
	writeRow("digraph enumtree {")

	// node primary markup
	writeRow(`node [color=Red,fontname=Courier,shape=circle]`)
	writeRow("edge [color=Blue, style=dashed]")

	// parse the nodes and map each node to its parent
	relations := make(map[int64]int64)
	for _, n := range t.nodes {
		writeRow("%v [label=prob_%v]", n.id, n.id)
		relations[n.id] = n.parent
	}

	// parse the edges
	for nodeID, parentID := range relations {

		// skip self-loops
		if nodeID == parentID {
			continue
		}

		writeRow("%v -> %v", parentID, nodeID)
	}

	writeRow("}")
}
