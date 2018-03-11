package ilp

import (
	"errors"
	"fmt"
	"log"
	"math"

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

func (p MILPproblem) Solve() (float64, []float64, error) {

	if len(p.integerVariables) != len(p.c) {
		panic("integerVariables vector is not same length as vector c")
	}

	// solve the initial LP relaxation
	z, x, err := lp.Simplex(p.c, p.A, p.b, 0, nil)

	// check if the problem has integrality constraints. If not, return the results of the LP relaxation.
	if !any(p.integerVariables) {
		return z, x, err
	}

	// check if initial LP relaxation is feasible considering the integrality constraints
	if isAllInteger(x...) {
		fmt.Println("Somehow the initial relaxation was feasible even in light of the integer constraints")
		return z, x, err
	}

	// Start the branch and bound procedure for this problem

	return 0, []float64{0}, errors.New("no solution")
}

// check whether the solution vector is feasible in light of the integrality constraints for each variable
func satisfiesIntegralityConstraints(constraints []bool, solution []float64) bool {
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

	// another option:
	// in == float64(int64(in))
}

type subProblem struct {
	// c, A, b represent the same as in the MILPproblem
	c []float64
	A *mat.Dense
	b []float64

	//
}
