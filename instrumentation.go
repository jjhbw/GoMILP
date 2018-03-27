package ilp

type bnbMiddleware interface {

	// Receives each subproblem solution and a corresponding decision
	ProcessDecision(solution, bnbDecision)
}

type dummyMiddleware struct{}

func (d dummyMiddleware) ProcessDecision(s solution, b bnbDecision) {
	return
}

// represents a node from the enumeration tree.
type node struct {
	id int64

	// objective function value
	z float64

	// intermediate solution
	x []float64
}

// convert a solution to a node, assiging an integer id.
func newNode(soln solution, identifier int64) node {
	return node{
		id: identifier,
		z:  soln.z,
		x:  soln.x,
	}
}
