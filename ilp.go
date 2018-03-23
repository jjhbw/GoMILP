package ilp

import (
	"errors"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

type milpProblem struct {
	// 	minimize c^T * x
	// s.t      G * x <= h
	//          A * x = b
	c []float64
	A *mat.Dense
	b []float64
	G *mat.Dense
	h []float64

	// which variables to apply the integrality constraint to. Should have same order as c.
	integralityConstraints []bool

	// which branching heuristic to use. Determines which integer variable is branched on at each split.
	// defaults to 0 == maxFun
	branchingHeuristic BranchHeuristic
}

type milpSolution struct {
	log      *logTree
	solution solution
}

var (
	INITIAL_RELAXATION_NOT_FEASIBLE = errors.New("initial relaxation is not feasible")
	NO_INTEGER_FEASIBLE_SOLUTION    = errors.New("no integer feasible solution found")
)

var (
	// problem-specific reasons why simplex-solving a problem can fail
	// these errors are expeced in a sense, do not warrant a panic, and correspond to a bnbDecision.
	expectedFailures = map[error]bnbDecision{
		lp.ErrInfeasible: SUBPROBLEM_IS_DEGENERATE,
		lp.ErrSingular:   SUBPROBLEM_NOT_FEASIBLE,
	}
)

func (p milpProblem) toInitialSubProblem() subProblem {
	return subProblem{
		c: p.c,
		A: p.A,
		b: p.b,
		G: p.G,
		h: p.h,
		integralityConstraints: p.integralityConstraints,

		// for the initial subproblem, there are no branch-and-bound-specific inequality constraints.
		bnbConstraints: []bnbConstraint{},
	}
}

// Argument workers specifies how many workers should be used for traversing the enumeration tree.
// This is mainly important from a space complexity point of view, as each worker is a potentially concurrent simplex algorithm.
func (p milpProblem) solve(workers int) (milpSolution, error) {
	if workers <= 0 {
		panic("number of workers may not be lower than zero")
	}

	if len(p.integralityConstraints) != len(p.c) {
		panic("integrality constraints vector is not same length as vector c")
	}

	// add the initial LP relaxation to the problem queue
	initialRelaxation := p.toInitialSubProblem()

	// Start the branch and bound procedure for this problem
	enumTree := newEnumerationTree(initialRelaxation)

	// start the branch and bound procedure, presenting the solution to the initial relaxation as a candidate
	incumbent, log := enumTree.startSearch(workers)

	if incumbent.err != nil {
		return milpSolution{}, incumbent.err
	}

	// check if the solution is feasible considering the integrality constraints
	if feasibleForIP(p.integralityConstraints, incumbent.x) {
		return milpSolution{}, NO_INTEGER_FEASIBLE_SOLUTION
	}

	return milpSolution{
		solution: incumbent,
		log:      log,
	}, nil

}
