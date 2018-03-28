package ilp

import (
	"fmt"
	"io"
)

type bnbMiddleware interface {

	// Receives a corresponding decision corresponding to a certain subproblem.
	ProcessDecision(solution, bnbDecision)

	// receives a new subproblem when it is created by the solver.
	NewProblem(subProblem)
}

type dummyMiddleware struct{}

func (d dummyMiddleware) ProcessDecision(s solution, b bnbDecision) {
	return
}

func (d dummyMiddleware) NewProblem(s subProblem) {
	return
}

type treeLogger struct {
	nodes map[int64]node
}

func newTreeLogger() *treeLogger {
	return &treeLogger{
		nodes: make(map[int64]node),
	}
}

// represents a node from the enumeration tree.
type node struct {
	id     int64
	parent int64

	// objective function value
	z float64

	// intermediate solution
	x []float64

	// whether the subproblem corresponding to this node has been solved
	solved bool

	// the branch-and-bound decision made by the solver corresponding to this decision
	// nil-valued if node is unsolved
	decision bnbDecision
}

// convert a subproblem to a node for the logging tree.
func newNode(p subProblem) node {
	return node{
		id:     p.id,
		parent: p.parent,

		// z, x, and decision are nil-valued at this point
	}
}

func (t *treeLogger) ProcessDecision(s solution, d bnbDecision) {
	node, found := t.nodes[s.problem.id]
	if !found {
		panic("tree logger: node not found in map. Not seen before?")
	}

	// update node values
	node.decision = d
	node.x = s.x
	node.z = s.z
	node.solved = true

	// reassign the node
	t.nodes[s.problem.id] = node
}

func (t *treeLogger) NewProblem(s subProblem) {
	if _, already := t.nodes[s.id]; already {
		panic("a node with this ID has already been logged")
	}
	t.nodes[s.id] = newNode(s)
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
	writeRow("node [fontname=Courier,shape=circle];")
	writeRow("edge [color=Blue, style=dashed];")

	// parse the nodes and map each node to its parent
	relations := make(map[int64]int64)
	for id, n := range t.nodes {
		color := "Red"
		if n.solved {
			color = "Green"
		}
		writeRow("%v [label=prob_%v,color=%v];", id, id, color)
		relations[id] = n.parent
	}

	// parse the edges
	for nodeID, parentID := range relations {

		// skip self-loops
		if nodeID == parentID {
			continue
		}

		writeRow("%v -> %v ;", parentID, nodeID)
	}

	writeRow("}")
}
