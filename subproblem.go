package ilp

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

type subProblem struct {

	// unique identifier for the subproblem
	id int64

	// id of the parent problem
	parent int64

	// These variables represent the same as in the MILPproblem and should not be modified.
	c []float64
	A *mat.Dense
	b []float64
	G *mat.Dense
	h []float64

	// integrality constraints, inherited from parent problem and should not be modified.
	integralityConstraints []bool

	// heuristic to determine variable to branch on. Inherited from parent and should not be modified.
	branchHeuristic BranchHeuristic

	// additional inequality constraints for branch-and-bound.
	// Each step down in the search procedure adds a constraint.
	bnbConstraints []bnbConstraint
}

type bnbConstraint struct {
	// the index of the variable that we branched on
	branchedVariable int

	// additions to make to the subProblem before solving
	hsharp float64
	gsharp []float64
}

type solution struct {
	problem *subProblem
	x       []float64
	z       float64
	err     error
}

// Retrieve all inequalities pertaining to this subProblem as a single G matrix and h vector.
// That means the inequalities of the original problem description and the ones added during the branch-and-bound procedure.
func (p subProblem) combineInequalities() (*mat.Dense, []float64) {

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

// Convert a problem with inequalities (G and h) to a problem with only nonnegative equalities (represented by matrix aNew and vector bNew) using slack variables
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

func (p subProblem) solve() solution {

	// get the inequality constraints
	G, h := p.combineInequalities()

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
		err:     err,
	}

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

	// increment the IDs of the subproblems accordingly
	p1.id++
	p2.id = p2.id + 2

	return
}

// inherit everything from the parent problem, but append a new bnb constraint using a variable index and a max value for this variable.
// Note that we also provide a multiplication factor for the to allow for sign changes.
func (p subProblem) getChild(branchOn int, factor float64, smallerOrEqualThan float64) subProblem {

	child := p.copy()
	newConstraint := bnbConstraint{
		branchedVariable: branchOn,
		hsharp:           smallerOrEqualThan,
		gsharp:           make([]float64, len(p.c)),
	}

	// point to the index of the variable to branch on
	newConstraint.gsharp[branchOn] = float64(factor)

	// add the constraint
	child.bnbConstraints = append(child.bnbConstraints, newConstraint)

	return child

}

// Creating child subProblems like this has non-trivial memory implications.
// Due to only containing reference types and pointers, the subProblem structs themselves are pretty lightweight.
// We try to avoid copying of subProblem field values, so the pointer values and the arrays underpinning the slices are reused a lot throughout the procedures.
// Make sure to run the race detector thoroughly after any modifications to this procedure.
// Note that copy assigns the same id integer to the daughter problem.
func (p *subProblem) copy() subProblem {
	new := subProblem{
		id:                     p.id,
		parent:                 p.id,
		c:                      p.c,
		A:                      p.A,
		b:                      p.b,
		G:                      p.G,
		h:                      p.h,
		bnbConstraints:         make([]bnbConstraint, len(p.bnbConstraints)),
		integralityConstraints: p.integralityConstraints,
	}

	// As the bnbConstraints slice is modified with each branch-and-bound node, we copy it to prevent race conditions occurring in subProblems further downstream
	copy(new.bnbConstraints, p.bnbConstraints)

	return new
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
