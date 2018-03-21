package ilp

// The logTree datastructure is used as a log of the branch-and-bound algorithms decisions.
// This code should not contain algorithm business logic to ensure loose coupling.
// Note that we don't want to store references to subproblem datastructures, as this would preclude GC for these potentially large structs.

type logTree struct {
	nodes []*node
}

type node struct {

	// a summary of the solution of this node
	x []float64
	z float64

	// the decision that took place at this node
	decision bnbDecision

	// the node's children, if any.
	children []*node
}

// Branch-and-bound decisions that can be made by the algorithm
type bnbDecision string

const (
	SUBPROBLEM_IS_DEGENERATE        bnbDecision = "subproblem contains a degenerate (singular) matrix"
	SUBPROBLEM_NOT_FEASIBLE         bnbDecision = "subproblem has no feasible solution"
	WORSE_THAN_INCUMBENT            bnbDecision = "worse than incumbent"
	BETTER_THAN_INCUMBENT_BRANCHING bnbDecision = "better than incumbent but infeasible, so branching"
	BETTER_THAN_INCUMBENT_FEASIBLE  bnbDecision = "better than incumbent and feasible, so replacing incumbent"
	INITIAL_RX_FEASIBLE_FOR_IP      bnbDecision = "initial relaxation is feasible for IP"
	INITIAL_RELAXATION_LEGAL        bnbDecision = "initial relaxation is legal"
)

// add a node to the logging tree.
// Note that we do not save a reference to the entire solution struct as this would be a potential GC nightmare.
func (t *logTree) addNode(s solution, decision bnbDecision) *node {
	n := &node{
		x:        s.x,
		z:        s.z,
		decision: decision,
		children: []*node{},
	}

	t.nodes = append(t.nodes, n)

	return n
}
