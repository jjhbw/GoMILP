package ilp

import (
	"errors"
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

// TODO: add more in-depth tests for the BNB routine: should properly find integer solutions.
// TODO: remove workaround for issue https://github.com/gonum/gonum/issues/441
// TODO: vendor dependencies
// TODO: in branched subproblems: intiate simplex at solution of parent? (using argument of lp.Simplex)
// TODO: visualising the enumeration tree?
// TODO: current calculation of DOF is more of a workaround around a gonum mat bug than a good calculation of DOF
// as it does not take into account whether the constraint equations are linearly independent.
// TODO: The used branching heuristic (selects a variable to branch on) is extremely dumb. Improve!

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

type MILPsolution struct {
	decisionLog []bnbStep
	solution    solution
}

// Branch-and-bound decisions
// TODO: using strings only for debugging, switch to int32 for smaller memory footprint on big problems
type bnbDecision string

const (
	SUBPROBLEM_NO_DOF               bnbDecision = "subproblem has no degrees of freedom"
	SUBPROBLEM_NOT_FEASIBLE         bnbDecision = "subproblem has no feasible solution"
	WORSE_THAN_INCUMBENT            bnbDecision = "worse than incumbent"
	BETTER_THAN_INCUMBENT_BRANCHING bnbDecision = "better than incumbent but infeasible, so branching"
	BETTER_THAN_INCUMBENT_FEASIBLE  bnbDecision = "better than incumbent and feasible, so replacing incumbent"
)

var (
	INITIAL_RELAXATION_NOT_FEASIBLE = errors.New("initial relaxation is not feasible")
	NO_INTEGER_FEASIBLE_SOLUTION    = errors.New("no integer feasible solution found")
	PROBLEM_HAS_NO_DOF              = errors.New("(sub)problem has DOF <= 0")
)

var (
	// problem-specific reasons why simplex-solving a problem can fail
	// these errors are expeced in a sense, do not warrant a panic, correspond to their respective bnbDecision.
	expectedFailures = map[error]bnbDecision{
		NO_INTEGER_FEASIBLE_SOLUTION: SUBPROBLEM_NOT_FEASIBLE,
		PROBLEM_HAS_NO_DOF:           SUBPROBLEM_NO_DOF,
	}
)

type bnbStep struct {
	solution         *solution
	currentIncumbent *solution
	decision         bnbDecision
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
	return nil, nil

}

// Sanity check for the problems dimensions
func sanityCheckDimensions(c []float64, A *mat.Dense, b []float64, G *mat.Dense, h []float64) error {
	if G != nil {
		if h == nil {
			return errors.New("h vector is nil while G matrix is provided")
		}

		rG, cG := G.Dims()
		if rG != len(h) {
			return errors.New("Number of rows in G matrix is not equal to length of h")
		}

		if cG != len(c) {
			return errors.New("Number of columns in G matrix is not equal to number of variables")
		}
	}

	if h != nil {
		if G == nil {
			return errors.New("G matrix is nil while h vector is provided")
		}
	}

	rA, cA := A.Dims()
	if rA != len(b) {
		return errors.New("Number of rows in A matrix is not equal to length of b")
	}

	if cA != len(c) {
		return errors.New("Number of columns in A matrix is not equal to number of variables")
	}

	return nil
}

// Convert a problem with inequalities (G and h) to a problem with only nonnegative equalities using slack variables
func convertToEqualities(c []float64, A *mat.Dense, b []float64, G *mat.Dense, h []float64) (cNew []float64, aNew *mat.Dense, bNew []float64) {

	//sanity checks
	if A == nil {
		panic("Provided pointer to A matrix is nil")
	}

	if G == nil {
		panic("Provided pointer to G matrix is nil")
	}

	if insane := sanityCheckDimensions(c, A, b, G, h); insane != nil {
		panic(insane)
	}

	// number of original variables
	nVar := len(c)

	// number of original constraints
	nCons := len(b)

	// number of inequalities to add
	nIneq := len(h)

	// new number of total variables
	nNewVar := nVar + nIneq

	// new total number of equality constraints
	nNewCons := len(b) + nIneq

	// construct new c
	cNew = make([]float64, nNewVar)
	copy(cNew, c)

	// add the slack variables to the objective function as zeroes
	copy(cNew[nVar:], make([]float64, nIneq))

	// concatenate the b and h vectors
	bNew = make([]float64, nNewCons)
	copy(bNew, b)
	copy(bNew[nCons:], h)

	// construct the new A matrix
	aNew = mat.NewDense(nNewCons, nNewVar, nil)

	// embed the original A matrix in the top left part of aNew, thus setting the original constraints
	aNew.Slice(0, nCons, 0, nVar).(*mat.Dense).Copy(A)

	// embed the G matrix into the new A, below the view of the old A.
	aNew.Slice(nCons, nNewCons, 0, nVar).(*mat.Dense).Copy(G)

	// diagonally fill the bottom-left part (next to G) with binary indicators of the slack variables
	bottomRight := aNew.Slice(nCons, nNewCons, nVar, nVar+nIneq).(*mat.Dense)
	for i := 0; i < nIneq; i++ {
		bottomRight.Set(i, i, 1)
	}

	// TODO: move to tests
	// sanity check the output dimensions
	if insane := sanityCheckDimensions(cNew, aNew, bNew, nil, nil); insane != nil {
		panic(insane)
	}

	return
}

// Get the degrees of freedom of a problem
func getDOF(c []float64, A mat.Matrix) int {
	rows, _ := A.Dims()
	return len(c) - rows
}

// TODO: WORKAROUND
// wrapper around Gonum's simplex algorithm to perform a preflight check on the DOF.
// if DOF <= 0, Simplex will panic at the BLAS level due to a bug in gonum's matrix implementation.
// see issue https://github.com/gonum/gonum/issues/441
func computeSimplexSimplex(c []float64, A mat.Matrix, b []float64, tol float64, initialBasic []int) (optF float64, optX []float64, err error) {
	if getDOF(c, A) <= 0 {
		return 0, nil, PROBLEM_HAS_NO_DOF
	}

	return lp.Simplex(c, A, b, tol, initialBasic)
}

func (p subProblem) solve() (solution, error) {

	// get the inequality constraints
	G, h := p.getInequalities()

	var z float64
	var x []float64
	var err error

	// if inequality constraints are presented, amend the problem with these.
	if G != nil {
		c, A, b := convertToEqualities(p.c, G, h, p.A, p.b)

		// fmt.Println("c:")
		// fmt.Println(c)
		// fmt.Println("A:")
		// fmt.Println(mat.Formatted(A))
		// fmt.Println("b:")
		// fmt.Println(b)

		z, x, err = computeSimplexSimplex(c, A, b, 0, nil)

		// take only the non-slack variables from the result.
		if err == nil && len(x) != len(p.c) {
			x = x[:len(p.c)]
		}

	} else {
		// fmt.Println("c:")
		// fmt.Println(p.c)
		// fmt.Println("A:")
		// fmt.Println(mat.Formatted(p.A))
		// fmt.Println("b:")
		// fmt.Println(p.b)

		z, x, err = computeSimplexSimplex(p.c, p.A, p.b, 0, nil)
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

	// build the subproblem that will explore the 'smaller or equal than' branch
	p1 = s.problem.getChild(branchOn, 1, math.Floor(currentCoeff))

	// formulate 'larger than' constraints of the branchpoint as 'smaller or equal than' by inverting the sign
	p2 = s.problem.getChild(branchOn, -1, -(math.Floor(currentCoeff) + 1))

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

// takes a solver failure and determines whether it warrants a panic or whether it is expected.
func translateSolverFailure(err error) bnbDecision {
	if err == lp.ErrInfeasible {
		return SUBPROBLEM_NOT_FEASIBLE
	}
	for failure, decision := range expectedFailures {
		if failure == err {
			return decision
		}
	}
	panic(err)
}

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

	// If no integrality constraints are present, we can return the initial solution as-is if it is feasible.
	// moreover, if the solution to the initial relaxation already satisfies all integrality constraints, we can present it as-is.
	if feasibleForIP(p.integralityConstraints, initialRelaxationSolution.x) {
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
			failure := translateSolverFailure(err)
			step.decision = failure

		// Note that the objective is a minimization.
		case incumbent.z <= candidate.z:
			// noop
			step.decision = WORSE_THAN_INCUMBENT

		case incumbent.z > candidate.z:
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
			// this should never happen and thus should never fail silently.
			panic("unexpected case: could not decide what to do with branched subproblem")

		}

		// save this step to the log
		steps = append(steps, step)

	}

	if !feasibleForIP(p.integralityConstraints, incumbent.x) {
		return MILPsolution{}, NO_INTEGER_FEASIBLE_SOLUTION
	}

	return MILPsolution{
		solution:    *incumbent,
		decisionLog: steps,
	}, nil

}

// check whether the solution vector is feasible in light of the integrality constraints for each variable
func feasibleForIP(constraints []bool, solution []float64) bool {
	if len(constraints) != len(solution) {
		panic(fmt.Sprint("constraints vector and solution vector not of equal size: ", constraints, solution))
	}
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
