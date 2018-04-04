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

func (p milpProblem) toInitialSubproblem() subProblem {
	// convert the inequalities (if any) to equalities
	cNew := p.c
	Anew := p.A
	bNew := p.b
	intNew := p.integralityConstraints

	if p.G != nil {
		cNew, Anew, bNew = convertToEqualities(p.c, p.A, p.b, p.G, p.h)

		// add 'false' integrality constraints to the created slack variables
		intNew = make([]bool, len(cNew))
		copy(intNew, p.integralityConstraints)

	}

	return subProblem{
		// the initial subproblem has 0 as identifier
		id: 0,

		c: cNew,
		A: Anew,
		b: bNew,
		integralityConstraints: intNew,

		// for the initial subproblem, there are no branch-and-bound-specific inequality constraints.
		bnbConstraints: []bnbConstraint{},
	}
}

// Argument workers specifies how many workers should be used for traversing the enumeration tree.
// This is mainly important from a space complexity point of view, as each worker is a potentially concurrent simplex algorithm.
func (p milpProblem) solve(ctx context.Context, workers int, instrumentation BnbMiddleware) (solution, error) {
	if workers <= 0 {
		panic("number of workers may not be lower than zero")
	}

	if len(p.integralityConstraints) != len(p.c) {
		panic("integrality constraints vector is not same length as vector c")
	}

	initialRelaxation := p.toInitialSubproblem()

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
		return val, timedOut
	}

	// Check if a nil solution has been returned
	if incumbent == nil {
		return solution{}, NO_INTEGER_FEASIBLE_SOLUTION
	}

	if incumbent.err != nil {
		return solution{}, incumbent.err
	}

	// remove the slack variables that were introduced by the conversion to standard form from the solution vector
	postprocessed := *incumbent
	postprocessed.x = postprocessed.x[:len(p.c)]

	return postprocessed, nil

}

//TODO: COPIED FROM GONUM FOR DEBUGGING: REMOVEME
// findLinearlyIndependnt finds a set of linearly independent columns of A, and
// returns the column indexes of the linearly independent columns.
func findLinearlyIndependent(A mat.Matrix) []int {
	m, n := A.Dims()
	idxs := make([]int, 0, m)
	columns := mat.NewDense(m, m, nil)
	newCol := make([]float64, m)
	// Walk in reverse order because slack variables are typically the last columns
	// of A.
	for i := n - 1; i >= 0; i-- {
		if len(idxs) == m {
			break
		}
		mat.Col(newCol, i, A)
		columns.SetCol(len(idxs), newCol)
		if len(idxs) == 0 {
			// A column is linearly independent from the null set.
			// If all-zero column of A are allowed, this code needs to be adjusted.
			idxs = append(idxs, i)
			continue
		}
		if mat.Cond(columns.Slice(0, m, 0, len(idxs)+1), 1) > 1e12 {
			// Not linearly independent.
			continue
		}
		idxs = append(idxs, i)
	}
	return idxs
}
