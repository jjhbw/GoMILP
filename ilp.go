package ilp

import (
	"context"
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

// Argument workers specifies how many workers should be used for traversing the enumeration tree.
// This is mainly important from a space complexity point of view, as each worker is a potentially concurrent simplex algorithm.
func (p milpProblem) solve(ctx context.Context, workers int, instrumentation BnbMiddleware) (milpSolution, error) {
	if workers <= 0 {
		panic("number of workers may not be lower than zero")
	}

	if len(p.integralityConstraints) != len(p.c) {
		panic("integrality constraints vector is not same length as vector c")
	}

	// preprocess the problem
	preprocessor := newPreprocessor()
	prepped := preprocessor.preSolve(p)

	initialRelaxation := prepped.toInitialSubproblem()

	// Start the branch and bound procedure for this problem
	enumTree := newEnumerationTree(initialRelaxation, instrumentation)

	// start the branch and bound procedure, presenting the solution to the initial relaxation as a candidate
	incumbent := enumTree.startSearch(ctx, workers)

	// if the solver timed out, we return that as an error, along with the best-effort incumbent solution.
	if timedOut := ctx.Err(); timedOut != nil {
		var val solution
		if incumbent != nil {
			val = *incumbent
		}
		return milpSolution{
			solution: val,
		}, timedOut
	}

	// Check if a nil solution has been returned
	if incumbent == nil {
		return milpSolution{}, NO_INTEGER_FEASIBLE_SOLUTION
	}

	if incumbent.err != nil {
		return milpSolution{}, incumbent.err
	}

	// undo all presolving operations
	postsolved := preprocessor.postSolve(*incumbent)

	return milpSolution{
		solution: postsolved,
	}, nil

}
