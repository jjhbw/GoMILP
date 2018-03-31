package ilp

import "gonum.org/v1/gonum/mat"

// TODO: see Andersen 1995 for a nice enumeration of simple presolving operations.

// TODO: decide for each presolve function where to apply it: at the enumeration tree root or on each subproblem
// TODO: decide for each step whether to store the decisions (removed vars)

// TODO: remove empty columns?

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
