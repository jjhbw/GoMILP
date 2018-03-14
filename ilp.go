package ilp

import (
	"errors"
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
	G *mat.Dense
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
	G *mat.Dense
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

func (p subProblem) getInequalities() (*mat.Dense, []float64) {

	if len(p.bnbConstraints) > 0 {
		// get the 'right sides' original problem inequality constraints
		h := p.h

		// build a matrix of all constraints originating from the branch-and-bound procedure
		var bnbGvects []float64
		for _, constr := range p.bnbConstraints {
			bnbGvects = append(bnbGvects, constr.gsharp...)

			// add the hsharp value to the h vector
			h = append(h, constr.hsharp)
		}
		bnbG := mat.NewDense(len(p.bnbConstraints), len(p.c), bnbGvects)

		// if the original problem did not contain inequality constraints, we return the bnb constraint matrix.
		if p.G == nil {
			return bnbG, h
		}

		// if for some magic reason the inequality constraint matrix is of zero-dimension, we can also return just the bnb constraints.
		if p.G.IsZero() {
			return bnbG, h
		}

		// Use stack to combine the branch-and-bound constraint matrix with the original problem inequality constraint matrix into G that will be used in the simplex
		// into a new matrix, which needs to be initialized in the exact shape we expect.
		// Note that this will place the bnb constraints in the higher indexed rows.
		origRows, _ := p.G.Dims()
		bnbRows, _ := bnbG.Dims()
		expectedRows := origRows + bnbRows

		// allocate a zero-valued new matrix of the given dimensions
		Gnew := mat.NewDense(expectedRows, len(p.c), nil)

		// stack the two matrices into this new matrix
		Gnew.Stack(p.G, bnbG)

		return Gnew, h
	}

	// if no constraints need to be added, return the original constraints.
	if p.G != nil {
		// copy the matrix, simultaneously casting to a concrete type
		return mat.DenseCopyOf(p.G), p.h
	}
	return nil, p.h

}

func (p subProblem) solve() (solution, error) {

	var c []float64
	var A *mat.Dense
	var b []float64

	// get the inequality constraints
	G, h := p.getInequalities()

	// if inequality constraints are presented (general form), convert the problem to standard form.
	if G != nil {
		c, A, b = lp.Convert(p.c, G, h, p.A, p.b)
	} else {
		c = p.c
		A = p.A
		b = p.b
	}

	// apply the simplex algorithm
	z, x, err := lp.Simplex(c, A, b, 0, nil)

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
func (s solution) branch(integralityConstraints []bool) (p1, p2 subProblem) {

	// get the variable to branch on by looking at which variables we branched on previously
	// if there are no branches yet, so we start at the first constrained variable
	branchOn := 0
	for i := range integralityConstraints {
		if integralityConstraints[i] {
			branchOn = i
		}
	}

	// if there are branches, we cycle through the variables starting from the last one we branched on
	// when we encounter the next variable with an integrality constraint, we pick that one to branch on.
	if len(s.problem.bnbConstraints) > 0 {

		// Get the last variable we branched.
		lastConstraint := s.problem.bnbConstraints[len(s.problem.bnbConstraints)-1]
		lastBranchedVariable := lastConstraint.branchedVariable

		// increment this variable until we encounter the next constrained variable or we reach the end of the variable vector.
		cursor := lastBranchedVariable
		for {
			if cursor == len(s.problem.c)-1 {
				// we bring the cursor back to the beginning
				cursor = -1
			}
			cursor++
			if integralityConstraints[cursor] {
				branchOn = cursor
				break
			}
		}

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

	// add the new constraint
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

// TODO: any logging for the tree visualisation should be done at the highest possible level. i.e. in this method
// TODO: better handling of errors

func (p MILPproblem) Solve() (MILPsolution, error) {

	if len(p.integralityConstraints) != len(p.c) {
		panic("integrality constraints vector is not same length as vector c")
	}

	// add the initial LP relaxation to the problem queue
	initialRelaxation := p.toInitialSubProblem()

	// solve the initial relaxation
	initialRelaxationSolution, err := initialRelaxation.solve()
	if err != nil {
		if err == lp.ErrInfeasible {
			// override the error message in case of infeasible initial relaxation for easier debugging
			return MILPsolution{}, INITIAL_RELAXATION_NOT_FEASIBLE
		}
		return MILPsolution{}, err
	}

	// if no integrality constraints are present, we can present the incumbent (initial relaxation) solution as-is
	if !any(p.integralityConstraints) {
		return MILPsolution{
			solution:    initialRelaxationSolution,
			decisionLog: nil,
		}, nil
	}

	// Start the branch and bound procedure for this problem
	var problemQueue []subProblem
	var steps []bnbStep
	var incumbent *solution

	// use the intial relaxation as the incumbent
	incumbent = &initialRelaxationSolution

	// branch the inital relaxation and add its children to the queue
	p1, p2 := incumbent.branch(p.integralityConstraints)
	problemQueue = append(problemQueue, p1, p2)

	fmt.Println(p)
	fmt.Println(initialRelaxation)
	fmt.Println(p1)
	fmt.Println(p2)

	for len(problemQueue) > 0 {

		// pop a problem from the queue
		var prob subProblem
		prob, problemQueue = problemQueue[0], problemQueue[1:]

		// solve the subproblem
		candidate, err := prob.solve()

		// store the state to be evaluated as a step
		step := bnbStep{
			solution:         &candidate,
			currentIncumbent: incumbent,
		}

		// decide on what to do with the solution:
		switch {

		case err != nil:
			// check if the subproblem was not feasible
			if err == lp.ErrInfeasible {
				step.decision = SUBPROBLEM_NOT_FEASIBLE
			} else {
				// any other error
				//TODO: clean this up
				fmt.Println(candidate.problem)
				panic(err)
			}

		case incumbent.z >= candidate.z:
			// noop
			step.decision = WORSE_THAN_INCUMBENT

		case incumbent.z < candidate.z:
			if feasibleForIP(p.integralityConstraints, candidate.x) {
				// candidate is an improvement over the incumbent
				incumbent = &candidate
				step.decision = BETTER_THAN_INCUMBENT_FEASIBLE
			} else {
				//candidate is an improvement over the incumbent, but not feasible.
				//branch and add the descendants of this candidate to the queue
				p1, p2 := candidate.branch(p.integralityConstraints)
				problemQueue = append(problemQueue, p1, p2)
				step.decision = BETTER_THAN_INCUMBENT_BRANCHING
			}

		default:
			// TODO: this should never happen
			panic("unexpected case")

		}

		// save this step to the log
		steps = append(steps, step)

	}

	return MILPsolution{
		solution:    *incumbent,
		decisionLog: steps,
	}, nil

}

type MILPsolution struct {
	decisionLog []bnbStep
	solution    solution
}

// Branch-and-bound decisions
// TODO: using strings only for debugging, switch to int32 for smaller memory footprint on big problems
type bnbDecision string

const (
	SUBPROBLEM_NOT_FEASIBLE         bnbDecision = "subproblem has no feasible solution"
	WORSE_THAN_INCUMBENT            bnbDecision = "worse than incumbent"
	BETTER_THAN_INCUMBENT_BRANCHING bnbDecision = "better than incumbent but infeasible, so branching"
	BETTER_THAN_INCUMBENT_FEASIBLE  bnbDecision = "better than incumbent and feasible, so replacing incumbent"
)

var (
	INITIAL_RELAXATION_NOT_FEASIBLE = errors.New("initial relaxation is not feasible")
)

type bnbStep struct {
	solution         *solution
	currentIncumbent *solution
	decision         bnbDecision
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
