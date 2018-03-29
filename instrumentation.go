package ilp

import (
	"fmt"
	"io"
)

type BnbMiddleware interface {

	// Receives a corresponding decision corresponding to a certain subproblem.
	ProcessDecision(solution, bnbDecision)

	// receives a new subproblem when it is created by the solver.
	NewSubProblem(subProblem)
}

type dummyMiddleware struct{}

func (d dummyMiddleware) ProcessDecision(s solution, b bnbDecision) {
	return
}

func (d dummyMiddleware) NewSubProblem(s subProblem) {
	return
}

type TreeLogger struct {
	nodes map[int64]node
}

func NewTreeLogger() *TreeLogger {
	return &TreeLogger{
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

func (t *TreeLogger) ProcessDecision(s solution, d bnbDecision) {
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

func (t *TreeLogger) NewSubProblem(s subProblem) {
	if _, already := t.nodes[s.id]; already {
		panic("a node with this ID has already been logged")
	}
	t.nodes[s.id] = newNode(s)
}

// takes an io.Writer to write the DOT-file visualisation of the processed enumeration tree to.
func (t *TreeLogger) ToDOT(out io.Writer) {

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
	writeRow("node [fontname=Courier,shape=rectangle];")
	writeRow("edge [color=Blue, style=dashed];")

	// parse the nodes and map each node to its parent
	relations := make(map[int64]int64)
	for id, n := range t.nodes {
		color := "Pink"
		label := "unsolved"
		if n.solved {
			tag := ""
			switch n.decision {
			case BETTER_THAN_INCUMBENT_FEASIBLE:
				color = "Green"
				tag = "new incumbent!"

			case SUBPROBLEM_NOT_FEASIBLE:
				color = "Red"
				tag = "infeasible"

			case WORSE_THAN_INCUMBENT:
				color = "Gray"
				tag = "worse"

			case BETTER_THAN_INCUMBENT_BRANCHING:
				color = "Black"
				tag = "branching"
			case SUBPROBLEM_IS_DEGENERATE:
				color = "Red"
				tag = "singular"

			default:
				color = "Red"
				tag = string(n.decision)
			}

			label = fmt.Sprintf("<Z=%.2f <BR /> id:%v <BR /> %v >", n.z, n.id, tag)
		}

		writeRow("%v [label=%v,color=%v];", id, label, color)
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
