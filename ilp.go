package ilp

import (
	"fmt"
	"log"
	"math"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

// TODO: write the branch and bound procedure
// TODO: Solver should output abstraction of the solution with some diagnostics
// TODO: try to formulate more advanced constraints, like sets of values instead of just integrality.
// Note that having integer sets as constraints is basically the same as having an integrality constraint + a <= and >= bound.
// Branching on this type of constraint can be optimized in a neat way (i.e. x>=0, x<=1, x<=0 ~-> x = 0)

type MILPproblem struct {
	// 	minimize c^T * x
	// s.t      G * x <= h
	//          A * x = b
	c []float64
	A *mat.Dense
	b []float64

	// additional inequality constraints:  G * x <= h
	// optional, may both be nil
	G *mat.Matrix
	h []float64

	// which variables to apply the integrality constraint to. Should have same order as c.
	integralityConstraints []bool
}

func (p MILPproblem) toInitialSubProblem() subProblem {
	return subProblem{
		c: p.c,
		A: p.A,
		b: p.b,
		G: p.G,
		h: p.h,

		// for the initial subproblem, there are no branch-and-bound-specific inequality constraints.
		bnbConstraints: []bnbConstraint{},
	}
}

type subProblem struct {
	// Variables represent the same as in the MILPproblem
	c []float64
	A *mat.Dense
	b []float64
	G *mat.Matrix
	h []float64

	// additional inequality constraints for branch-and-bound
	bnbConstraints []bnbConstraint
}

type bnbConstraint struct {
	// the index of the variable that we branched on
	branchedVariable int

	// additions to make to the subProblem before solving
	hsharp float64
	gsharp []float64
}

func (p subProblem) solve() (solution, error) {
	var z float64
	var x []float64
	var err error

	// TODO: if any additional branch-and-bound constraints are present, add these to the inequality constraints
	// TODO: note that this can get tricky as we dont want to MODIFY any of the matrices

	// if inequality constraints are presented in general form, convert the problem to standard form.
	if p.G == nil || p.h == nil {
		z, x, err = lp.Simplex(p.c, p.A, p.b, 0, nil)
	} else {
		c, a, b := lp.Convert(p.c, *p.G, p.h, p.A, p.b)
		z, x, err = lp.Simplex(c, a, b, 0, nil)
	}

	if err != nil {
		return solution{}, err
	}

	return solution{
		problem: &p,
		x:       x,
		z:       z,
	}, err

}

type solution struct {
	problem *subProblem
	x       []float64
	z       float64
}

// branch the solution into two subproblems that have an added constraint on a particular variable in a particular direction.
// Which variable we branch on is controlled using the variable index specified in the branchOn argument.
// The integer value on which to branch is inferred from the parent solution.
// e.g. if this is the first time the problem has branched: create two new problems with new constraints on variable x1, etc.
func (s solution) branch() (p1, p2 subProblem) {

	// get the variable to branch on by looking at which variables we branched on previously
	// if there are no branches yet, so we start at the first variable (with index = 0)
	branchOn := 0
	if len(s.problem.bnbConstraints) > 0 {
		// Determine which variable to branch on next based on the last variable we branched on and the number of variables.
		lastConstraint := s.problem.bnbConstraints[len(s.problem.bnbConstraints)-1]
		lastBranchedVariable := lastConstraint.branchedVariable
		if lastBranchedVariable < len(s.problem.c)-1 {
			branchOn = lastBranchedVariable + 1
		}

		// if we already branched on all variables, we start back at the first.
	}

	// Formulate the right constraints for this variable, based on its coefficient estimated by the current solution.
	currentCoeff := s.x[branchOn]

	// build the subproblem that will explore the 'smaller than' branch
	p1 = s.problem.getChild(branchOn, 1, math.Floor(currentCoeff))

	// formulate 'larger than' constraints of the branchpoint as 'smaller than' by inverting the sign
	p2 = s.problem.getChild(branchOn, -1, -math.Ceil(currentCoeff))

	return
}

// inherit everything from the parent problem, but append a new bnb constraint using a variable index and a max value for this variable.
// Note that we also provide a multiplication factor for the to allow for sign changes
func (p subProblem) getChild(branchOn int, factor float64, smallerOrEqualThan float64) subProblem {

	child := p
	newConstraint := bnbConstraint{
		branchedVariable: branchOn,
		hsharp:           smallerOrEqualThan,
		gsharp:           make([]float64, len(p.c)),
	}

	// point to the index of the variable to branch on
	newConstraint.gsharp[branchOn] = float64(factor)

	child.bnbConstraints = append(child.bnbConstraints, newConstraint)

	return child

}

func any(in []bool) bool {
	for _, x := range in {
		if x {
			return true
		}
	}
	return false
}

func (p MILPproblem) Solve() (float64, []float64, error) {

	if len(p.integralityConstraints) != len(p.c) {
		panic("integrality constraints vector is not same length as vector c")
	}

	// check if the problem has integrality constraints. If not, return the results of the LP relaxation.
	if !any(p.integralityConstraints) {
		return lp.Simplex(p.c, p.A, p.b, 0, nil)
	}

	// Start the branch and bound procedure for this problem
	var incumbent *solution
	var problemQueue []subProblem

	// add the initial LP relaxation to the problem queue
	initialRelaxation := p.toInitialSubProblem()
	problemQueue = append(problemQueue, initialRelaxation)

	for len(problemQueue) > 0 {

		// pop a problem from the queue
		var prob subProblem
		prob, problemQueue = problemQueue[0], problemQueue[1:]

		// solve the subproblem
		candidate, err := prob.solve()

		// check if initial LP relaxation has failed (e.g. because it is not feasible)
		if err != nil {
			return 0, nil, err
		}

		// decide on what to do with the solution:
		switch {
		// solution is not feasible
		case err == lp.ErrInfeasible:
			// noop

		case incumbent.z >= candidate.z:
			// noop

		case incumbent.z < candidate.z:
			if feasibleForIP(p.integralityConstraints, candidate.x) {
				// candidate is an improvement over the incumbent
				incumbent = &candidate
			} else {
				//candidate is an improvement over the incumbent, but not feasible.
				//TODO: branch and add the descendants of this candidate to the queue

			}

		}
	}

	//TODO: try to retain the information as to why the incumbent is nil at this point in the algorithm
	if incumbent == nil {
		return 0, nil, nil
	}

	return incumbent.z, incumbent.x, nil

}

// check whether the solution vector is feasible in light of the integrality constraints for each variable
func feasibleForIP(constraints []bool, solution []float64) bool {
	for i := range solution {
		if constraints[i] {
			if !isAllInteger(solution[i]) {
				return false
			}
		}
	}
	return true
}

func isAllInteger(in ...float64) bool {
	for _, k := range in {
		if !(k == math.Trunc(k)) {
			return false
		}
	}
	return true
}

// ExampleSimplex smoke tests the Gonum simplex lp solver and serves as an example.
func ExampleSimplex() {
	// standard form:
	// 	minimize	c^T x
	// s.t. 		A * x = b
	// 				x >= 0 .

	// this example solves the following problem:
	// Minimize Z = -1x1 + -2x2 + 0x3 + 0x4
	// Subject to:
	//		-1x1 	+ 2x2 	+ 1x3 	+ 0x4 	= 4
	//		3x1 	+ 1x2 	+ 0x3 	+ 1x4 	= 9

	c := []float64{-1, -2, 0, 0}
	A := mat.NewDense(2, 4, []float64{
		-1, 2, 1, 0,
		3, 1, 0, 1,
	})
	b := []float64{4, 9}

	z, x, err := lp.Simplex(c, A, b, 0, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("opt: %v\n", z)
	fmt.Printf("x: %v\n", x)
	// Output:
	// z: -8
	// x: [2 3 0 0]
}
