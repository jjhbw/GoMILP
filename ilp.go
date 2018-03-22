package ilp

import (
	"errors"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

// TODO: add more diverse MILP test cases with known solutions for the BNB routine.
// TODO: primal vs dual simplex; any benefit?
// TODO: how to deal with matrix degeneracy in subproblems? Currently handled the same way as infeasible subproblems.
// TODO: in branched subproblems: intiate simplex at solution of parent? (using argument of lp.Simplex)
// TODO: does fiddling with the simplex tolerance value improve outcomes?
// TODO: Currently implemented only the simplest branching heuristics. Room for improvement.
// TODO: ? if branching yields an infeasible or otherwise unsolveable problem, try with another branching heuristic or use the second-best option.
// TODO: also fun: linear program preprocessing (MATLAB docs: https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html#btv20av)
// TODO: Queue is currently FIFO. For depth-first exploration, we should go with a LIFO queue.
// TODO: Add heuristic determining which node gets explored first (as we are using depth-first search) https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html?s_tid=gn_loc_drop#btzwtmv

const (
	// TODO: move to argument
	N_WORKERS = 2
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

func (p milpProblem) solve() (milpSolution, error) {

	if len(p.integralityConstraints) != len(p.c) {
		panic("integrality constraints vector is not same length as vector c")
	}

	// add the initial LP relaxation to the problem queue
	initialRelaxation := p.toInitialSubProblem()

	// Start the branch and bound procedure for this problem
	enumTree := newEnumerationTree(initialRelaxation)

	// start the branch and bound procedure, presenting the solution to the initial relaxation as a candidate
	incumbent, log := enumTree.startSearch(N_WORKERS)

	if incumbent.err == INITIAL_RELAXATION_NOT_FEASIBLE {
		return milpSolution{}, INITIAL_RELAXATION_NOT_FEASIBLE
	}

	// check if the solution is feasible considering the integrality constraints
	if incumbent.err != nil || !feasibleForIP(p.integralityConstraints, incumbent.x) {
		return milpSolution{}, NO_INTEGER_FEASIBLE_SOLUTION
	}

	return milpSolution{
		solution: incumbent,
		log:      log,
	}, nil

}
