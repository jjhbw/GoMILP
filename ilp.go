package ilp

import (
	"errors"
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

// TODO: add more diverse MILP test cases with known solutions for the BNB routine.
// TODO: primal vs dual simplex; any benefit?
// TODO: how to deal with matrix degeneracy in subproblems? Currently handled the same way as infeasible subproblems.
// TODO: in branched subproblems: intiate simplex at solution of parent? (using argument of lp.Simplex)
// TODO: does fiddling with the simplex tolerance value improve outcomes?
// TODO: visualising the enumeration tree?
// TODO: Currently implemented only the simplest branching heuristics. Room for improvement.
// TODO: ? if branching yields an infeasible or otherwise unsolveable problem, try with another branching heuristic or use the second-best option.
// TODO: also fun: linear program preprocessing (MATLAB docs: https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html#btv20av)
// TODO: Queue is currently FIFO. For depth-first exploration, we should go with a LIFO queue.
// TODO: Add heuristic determining which node gets explored first (as we are using depth-first search) https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html?s_tid=gn_loc_drop#btzwtmv

type MILPproblem struct {
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

type MILPsolution struct {
	decisionLog []bnbStep
	solution    solution
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
)

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
		integralityConstraints: p.integralityConstraints,

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

	// integrality constraints, inherited from parent.
	integralityConstraints []bool

	// additional inequality constraints for branch-and-bound
	bnbConstraints []bnbConstraint

	// heuristic to determine variable to branch on
	branchHeuristic BranchHeuristic
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
	// Either G or A needs to be provided
	if G == nil && A == nil {
		return errors.New("No constraint matrices provided")
	}

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

	if A != nil {
		rA, cA := A.Dims()
		if rA != len(b) {
			return errors.New("Number of rows in A matrix is not equal to length of b")
		}

		if cA != len(c) {
			return errors.New("Number of columns in A matrix is not equal to number of variables")
		}
	}

	if b != nil {
		if A == nil {
			return errors.New("A matrix is nil while b vector is provided")
		}
	}

	return nil
}

// Convert a problem with inequalities (G and h) to a problem with only nonnegative equalities using slack variables
func convertToEqualities(c []float64, A *mat.Dense, b []float64, G *mat.Dense, h []float64) (cNew []float64, aNew *mat.Dense, bNew []float64) {

	//sanity checks
	// A may be nil (if it is, we can initiate a new one),
	// but as this function's explicit purpose is converting inequalities, G may not be nil.
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

	// if A is not nil, embed the original A matrix in the top left part of aNew, thus setting the original constraints
	if A != nil {
		aNew.Slice(0, nCons, 0, nVar).(*mat.Dense).Copy(A)
	}

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

func (p subProblem) solve() (solution, error) {

	// get the inequality constraints
	G, h := p.getInequalities()

	var z float64
	var x []float64
	var err error

	// if inequality constraints are presented, amend the problem with these.
	if G != nil {
		c, A, b := convertToEqualities(p.c, p.A, p.b, G, h)

		z, x, err = lp.Simplex(c, A, b, 0, nil)

		// take only the non-slack variables from the result.
		if err == nil && len(x) != len(p.c) {
			x = x[:len(p.c)]
		}

	} else {
		z, x, err = lp.Simplex(p.c, p.A, p.b, 0, nil)
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

	// select variable to branch on based on the provided heuristic method
	branchOn := 0
	switch s.problem.branchHeuristic {
	case BRANCH_MAXFUN:
		branchOn = maxFunBranchPoint(s.problem.c, s.problem.integralityConstraints)

	case BRANCH_MOST_INFEASIBLE:
		branchOn = mostInfeasibleBranchPoint(s.problem.c, s.problem.integralityConstraints)

	case BRANCH_NAIVE:
		branchOn = s.naiveBranchPoint()

	default:
		panic("provided branching heuristic config variable unknown")
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
			solution: initialRelaxationSolution,
			decisionLog: []bnbStep{{
				solution:         &initialRelaxationSolution,
				currentIncumbent: nil,
				decision:         INITIAL_RX_FEASIBLE_FOR_IP,
			}},
		}, nil
	}

	// Start the branch and bound procedure for this problem
	var problemQueue []subProblem
	var steps []bnbStep
	var incumbent *solution

	// use the intial relaxation as the incumbent
	incumbent = &initialRelaxationSolution

	// branch the inital relaxation and add its children to the queue
	p1, p2 := incumbent.branch()
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
				p1, p2 := candidate.branch()
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
