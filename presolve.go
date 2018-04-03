package ilp

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// TODO: see Andersen 1995 for a nice enumeration of simple presolving operations.

// TODO: remove empty columns

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

// store all post-solving operations that bring the solution back to its input shape.
type preProcessor struct {
	undoers []undoer
}

type undoer func(solution) solution

func newPreprocessor() *preProcessor {
	return &preProcessor{}
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

//TODO: copies data; ok?
func removeEmptyRows(A *mat.Dense, b []float64) (*mat.Dense, []float64) {

	// Remove all rows of the equality constraint matrix that are empty (i.e. all values in row are 0)
	aRows, aCols := A.Dims()
	var nonEmptyRows []int
	for i := 0; i < aRows; i++ {

		// find nonzero values in row
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

	// if no empty rows where found, we return a copy of A
	if len(nonEmptyRows) == aRows {
		fmt.Println("Preprocessor: No empty rows found")

		bNew := make([]float64, aRows)
		copy(bNew, b)
		return mat.DenseCopyOf(A), bNew
	}

	//TODO: remove zero rows
	ANew := dropRows(A, nonEmptyRows)

	// update the new b vector by index
	var bNew []float64
	for _, r := range nonEmptyRows {
		bNew = append(bNew, b[r])
	}

	// make a new matrix containing only the non-empty rows
	if len(nonEmptyRows) == 0 {
		panic("all rows of A are empty")
	}

	fmt.Printf("removed %v redundant rows\n", aRows-len(nonEmptyRows))

	return ANew, bNew
}

func dropRows(m mat.Matrix, keep []int) (new *mat.Dense) {
	_, cols := m.Dims()

	var newData []float64

	for _, r := range keep {
		// get the row of the matrix object
		row := make([]float64, cols)
		mat.Row(row, r, m)
		newData = append(newData, row...)
	}

	new = mat.NewDense(len(keep), cols, newData)

	return
}

// func (prepper *preProcessor) removeFixedVariables(c []float64, A *mat.Dense, ints []bool) (cNew []float64, Anew *mat.Dense, intNew []bool) {

// 	// for each column in A, check whether it has only one nonzero element.
// 	// If it has only one nonzero element, we save that row
// 	aR, aC := A.Dims()

// 	for j := 0; j < aC; j++ {
// 		nonzero := 0
// 		latestNonzero := -1
// 		for i := 0; i < aC; i++ {
// 			if A.At(i, j) != 0 {
// 				nonzero++
// 				latestNonzero = i
// 			}
// 		}
// 		if nonzero == 1 {

// 		}
// 	}

// }

// identify any empty columns, returning an indicator vector.
func findEmptyColumns(m mat.Matrix) (emptyCols []bool) {
	aR, aC := m.Dims()

	emptyCols = make([]bool, aC)

	for j := 0; j < aC; j++ {
		nonzero := false

	nonzeroFinder:
		for i := 0; i < aR; i++ {
			if m.At(i, j) != 0 {
				nonzero = true
				break nonzeroFinder
			}
		}

		if !nonzero {
			emptyCols[j] = true
		}
	}

	return
}

//TODO: currently only processes empty columns with cj = 0
func (prepper *preProcessor) processEmptyColumns(A *mat.Dense, c []float64, ints []bool) (Aslim *mat.Dense, cNew []float64, intsNew []bool) {

	emptyIndicator := findEmptyColumns(A)

	if !any(emptyIndicator) {
		fmt.Println("no empty columns to remove")
		return A, c, ints
	}

	var toKeep []int
	for i, empty := range emptyIndicator {
		// Because variable xj does not appear in the objective function c and matrix A, it has no influence on the problem,
		// thus column j can be removed from the LO problem.
		// The solution of problem  is not affected by the removal of an empty column.
		//TODO: During the postsolve procedure, the value of xj can be set to any value, satisfying lj ≤ xj ≤ uj
		if empty {
			if c[i] == 0 {
				//remove
				continue
			}
		}
		toKeep = append(toKeep, i)

	}

	// remove empty columns from A by feeding the transpose to the row dropper.
	Aslim = dropRows(A.T(), toKeep).T().(mat.Transpose).Untranspose().(*mat.Dense)

	// remove empty columns from and integrality constraints
	for _, x := range toKeep {
		cNew = append(cNew, c[x])
		intsNew = append(intsNew, ints[x])
	}

	fmt.Println("number of removed empty columns: ", len(emptyIndicator)-len(toKeep))

	return

}

func any(in []bool) bool {
	for _, x := range in {
		if x {
			return true
		}
	}
	return false
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

	A, c, integrality = prepper.processEmptyColumns(A, c, integrality)

	preppedProblem := preProcessedProblem{
		c: c,
		A: A,
		b: b,
		integralityConstraints: integrality,
	}

	ar, ac := preppedProblem.A.Dims()
	fmt.Printf("Dims of A matrix in preprocessed standard-form initial relaxation: %v rows, %v columns \n", ar, ac)

	return preppedProblem
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
