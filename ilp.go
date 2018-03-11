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
	integerVariables []bool
}

func (p MILPproblem) toInitialSubProblem() subProblem {
	return subProblem{
		c: p.c,
		A: p.A,
		b: p.b,
		G: p.G,
		h: p.h,
	}
}

type subProblem struct {
	// c, A, b represent the same as in the MILPproblem
	c []float64
	A *mat.Dense
	b []float64

	// additional, optional inequality constraints:  G * x <= h
	G *mat.Matrix
	h []float64
}

type Solution struct {
	problem *subProblem
	x       []float64
	z       float64
}

func (p subProblem) solve() (Solution, error) {
	var z float64
	var x []float64
	var err error

	// if inequality constraints are presented in general form, convert the problem to standard form.
	if p.G == nil || p.h == nil {
		z, x, err = lp.Simplex(p.c, p.A, p.b, 0, nil)
	} else {
		c, a, b := lp.Convert(p.c, *p.G, p.h, p.A, p.b)
		z, x, err = lp.Simplex(c, a, b, 0, nil)
	}

	if err != nil {
		return Solution{}, err
	}

	return Solution{
		problem: &p,
		x:       x,
		z:       z,
	}, err

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

	// check if the problem has integrality constraints. If not, return the results of the LP relaxation.
	if !any(p.integerVariables) {
		return lp.Simplex(p.c, p.A, p.b, 0, nil)
	}

	// // check if initial LP relaxation is feasible considering the integrality constraints
	// if satisfiesIntegralityConstraints(p.integerVariables, x) {
	// 	fmt.Println("Somehow the initial relaxation was feasible even in light of the integrality constraints")
	// 	return z, x, err
	// }

	// TODO: Start the branch and bound procedure for this problem
	var incumbent Solution
	var problemQueue []subProblem

	// add the initial LP relaxation to the problem queue
	initialRelaxation := p.toInitialSubProblem()
	problemQueue = append(problemQueue, initialRelaxation)

mainLoop:
	for len(problemQueue) > 0 {

		// pop a problem from the queue
		var prob subProblem
		prob, problemQueue = problemQueue[0], problemQueue[1:]

		// solve the subproblem
		sol, err := prob.solve()

		break mainLoop
	}

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
}

// func newSubProblem(c []float64, A *mat.Dense,b []float64) subProblem {

// }

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
