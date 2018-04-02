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

func (p milpProblem) toInitialSubProblem() subProblem {

	// get the standard form representation of the problem
	c, A, b, integrality := p.toStandardForm()

	return subProblem{
		// the initial subproblem has 0 as identifier
		id: 0,

		c: c,
		A: A,
		b: b,
		integralityConstraints: integrality,

		// for the initial subproblem, there are no branch-and-bound-specific inequality constraints.
		bnbConstraints: []bnbConstraint{},
	}
}

func (p milpProblem) toStandardForm() (cNew []float64, Anew *mat.Dense, bNew []float64, intNew []bool) {

	// convert the inequalities (if any) to equalities
	cNew = p.c
	Anew = p.A
	bNew = p.b
	intNew = p.integralityConstraints

	if p.G != nil {
		cNew, Anew, bNew = convertToEqualities(p.c, p.A, p.b, p.G, p.h)

		// add 'false' integrality constraints to the created slack variables
		intNew = make([]bool, len(cNew))
		copy(intNew, p.integralityConstraints)
		return
	}

	return

}

// Argument workers specifies how many workers should be used for traversing the enumeration tree.
// This is mainly important from a space complexity point of view, as each worker is a potentially concurrent simplex algorithm.
func (p milpProblem) solve(ctx context.Context, workers int, instrumentation BnbMiddleware) (milpSolution, error) {
	if workers <= 0 {
		panic("number of workers may not be lower than zero")
	}

	if len(p.integralityConstraints) != len(p.c) {
		panic("integrality constraints vector is not same length as vector c")
	}

	// add the initial LP relaxation to the problem queue
	initialRelaxation := p.toInitialSubProblem()

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

	// TODO: replace with formal postsolve routine to undo all presolver operations.
	// map the solution back to its original shape (i.e. remove slack variables)
	remapped := solution{
		problem: incumbent.problem,
		x:       incumbent.x[:len(p.c)],
		z:       incumbent.z,
		err:     incumbent.err,
	}

	return milpSolution{
		solution: remapped,
	}, nil

}
