package ilp

import (
	"fmt"
	"log"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

// ExampleSimplex smoke tests the Gonum simplex lp solver and serves as an example.
func ExampleSimplex() {

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

type MILPproblem struct {
	// see ExampleSimplex for an example on what c, A, and b represent.
	c []float64
	A *mat.Dense
	b []float64

	// which variables to apply the integrality constraint to. Same order as c.
	integerVariables []bool
}

func any(in []bool) bool {
	for _, x := range in {
		if x {
			return true
		}
	}
	return false
}

func (p MILPproblem) Solve() (z float64, x []float64, err error) {

	if len(p.integerVariables) != len(p.c) {
		panic("integerVariables vector is not same length as vector c")
	}

	if !any(p.integerVariables) {
		z, x, err := lp.Simplex(p.c, p.A, p.b, 0, nil)
		return z, x, err
	}

	return 0, []float64{0}, nil
}
