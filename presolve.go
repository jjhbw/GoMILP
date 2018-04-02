package ilp

import (
	"gonum.org/v1/gonum/mat"
)

// TODO: see Andersen 1995 for a nice enumeration of simple presolving operations.

// TODO: decide for each presolve function where to apply it: at the enumeration tree root or on each subproblem
// TODO: decide for each step whether to store the decisions (removed vars)
// TODO: store 'postsolving' functions to undo each presolving operation

type preProcessedProblem struct {
	c []float64
	A *mat.Dense
	b []float64

	// which variables to apply the integrality constraint to. Should have same order as c.
	integralityConstraints []bool
}

func (p preProcessedProblem) toInitialSubproblem() subProblem {

	return subProblem{
		// the initial subproblem has 0 as identifier
		id: 0,

		c: p.c,
		A: p.A,
		b: p.b,
		integralityConstraints: p.integralityConstraints,

		// for the initial subproblem, there are no branch-and-bound-specific inequality constraints.
		bnbConstraints: []bnbConstraint{},
	}
}

// TODO: store all post-solving operations in a stack.
type preProcessor struct {
	undoers []undoer
}

type undoer func(solution) solution

func newPreprocessor() *preProcessor {
	return &preProcessor{}
}

// TODO: remove empty columns?
//TODO: copies data; ok?
func removeEmptyRows(A *mat.Dense, b []float64) (*mat.Dense, []float64) {

	// Remove all rows of the equality constraint matrix that are empty (i.e. all values in row are 0)
	aRows, aCols := A.Dims()
	var nonEmptyRows []int
	for i := 0; i < aRows; i++ {

		// find nonzero values
		nonzero := false
	jloop:
		for j := 0; j < aCols; j++ {
			if A.At(i, j) != 0 {
				nonzero = true
				break jloop
			}
		}

		if nonzero {
			nonEmptyRows = append(nonEmptyRows, i)
		}

	}

	// make a new matrix containing only the non-empty rows
	if len(nonEmptyRows) == 0 {
		panic("all rows of A are empty")
	}

	// if no empty rows where found, we return a copy of A
	if len(nonEmptyRows) == aRows {
		bNew := make([]float64, aRows)
		copy(bNew, b)
		return mat.DenseCopyOf(A), bNew
	}

	var newAData []float64
	var bNew []float64
	for _, r := range nonEmptyRows {
		//  RawRowView returns a slice backed by the same array as backing the receiver.
		newAData = append(newAData, A.RawRowView(r)...)

		// update the new b vector by index
		bNew = append(bNew, b[r])

	}

	ANew := mat.NewDense(len(nonEmptyRows), aCols, newAData)

	return ANew, bNew
}

// wraps the convertToEqualities function to convert a milpProblem to standard form (converting inequalities to equalities)
func (prepper *preProcessor) toStandardForm(p milpProblem) (cNew []float64, Anew *mat.Dense, bNew []float64, intNew []bool) {

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

		// create the corresponding undoer map the solution back to its original shape (i.e. remove slack variables)
		prepper.addUndoer(func(s solution) solution {
			return solution{
				x:       s.x[:len(p.c)],
				z:       s.z,
				err:     s.err,
				problem: s.problem,
			}
		})
		return
	}

	return

}

func (prepper *preProcessor) addUndoer(u undoer) {
	prepper.undoers = append(prepper.undoers, u)
}

// TODO: take care not to modify the original milpProblem matrices
func (prepper *preProcessor) preSolve(p milpProblem) preProcessedProblem {

	// get the standard form representation of the problem
	c, A, b, integrality := prepper.toStandardForm(p)

	// removing empty rows does not require any postsolve operations to be stored
	A, b = removeEmptyRows(A, b)

	return preProcessedProblem{
		c: c,
		A: A,
		b: b,
		integralityConstraints: integrality,
	}
}

func (prepper *preProcessor) postSolve(s solution) solution {

	sol := s
	// walk the slice from the last to the first element (use it as a LIFO queue)
	n := len(prepper.undoers)
	for i := n - 1; i == 0; i-- {
		undo := prepper.undoers[i]
		sol = undo(sol)
	}

	return sol
}
