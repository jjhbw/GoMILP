package ilp

import (
	"testing"

	"gonum.org/v1/gonum/mat"

	"github.com/stretchr/testify/assert"
)

// a simple minimization (the default) case with one inequality and no integrality constraints
func TestProblem_toSolveableA(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable("v1").SetCoeff(-1)
	v2 := prob.AddVariable("v2").SetCoeff(-2)
	v3 := prob.AddVariable("v3").SetCoeff(1)
	v4 := prob.AddVariable("v4").SetCoeff(3)

	// add the equality constraints
	prob.AddConstraint().AddExpression(1, v1).EqualTo(5)
	prob.AddConstraint().AddExpression(3, v2).EqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).EqualTo(2)
	prob.AddConstraint().AddExpression(1, v4).SmallerThanOrEqualTo(2)

	solveable := prob.toSolveable()
	expected := MILPproblem{
		c: []float64{-1, -2, 1, 3},
		A: mat.NewDense(3, 4, []float64{
			1, 0, 0, 0,
			0, 3, 0, 0,
			0, 0, 1, 0,
		}),
		b: []float64{5, 2, 2},
		G: mat.NewDense(1, 4, []float64{
			0, 0, 0, 1,
		}),
		h: []float64{2},
		integralityConstraints: []bool{false, false, false, false},
	}

	//Note:  do not compare pointers
	assert.Equal(t, expected, *solveable)
}

// A minimization: no inequalities and 2 integrality constraints
func TestProblem_toSolveableB(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable("v1").SetCoeff(-1)
	v2 := prob.AddVariable("v2").IsInteger().SetCoeff(-2)
	v3 := prob.AddVariable("v3").IsInteger().SetCoeff(1)

	// add the equality constraints
	prob.AddConstraint().AddExpression(1, v1).EqualTo(5)
	prob.AddConstraint().AddExpression(3, v2).EqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).EqualTo(2)

	solveable := prob.toSolveable()
	expected := MILPproblem{
		c: []float64{-1, -2, 1},
		A: mat.NewDense(3, 3, []float64{
			1, 0, 0,
			0, 3, 0,
			0, 0, 1,
		}),
		b: []float64{5, 2, 2},
		G: nil,
		h: nil,
		integralityConstraints: []bool{false, true, true},
	}

	//Note:  do not compare pointers
	assert.Equal(t, expected, *solveable)
}

// A maximization: no inequalities and 2 integrality constraints
func TestProblem_toSolveableC(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable("v1").SetCoeff(-1)
	v2 := prob.AddVariable("v2").SetCoeff(-2).IsInteger()
	v3 := prob.AddVariable("v3").SetCoeff(1).IsInteger()

	// add the equality constraints
	prob.AddConstraint().AddExpression(1, v1).EqualTo(5)
	prob.AddConstraint().AddExpression(3, v2).EqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).EqualTo(2)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.toSolveable()
	expected := MILPproblem{
		c: []float64{1, 2, -1},
		A: mat.NewDense(3, 3, []float64{
			1, 0, 0,
			0, 3, 0,
			0, 0, 1,
		}),
		b: []float64{5, 2, 2},
		G: nil,
		h: nil,
		integralityConstraints: []bool{false, true, true},
	}

	//Note:  do not compare pointers
	assert.Equal(t, expected, *solveable)
}

// constraints involving multiple variables
func TestProblem_toSolveableD(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable("v1").SetCoeff(-1)
	v2 := prob.AddVariable("v2").SetCoeff(-2).IsInteger()
	v3 := prob.AddVariable("v3").SetCoeff(1).IsInteger()

	// add the equality constraints
	prob.AddConstraint().AddExpression(1, v1).AddExpression(1, v2).EqualTo(5)
	prob.AddConstraint().AddExpression(3, v2).EqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).EqualTo(2)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.toSolveable()
	expected := MILPproblem{
		c: []float64{1, 2, -1},
		A: mat.NewDense(3, 3, []float64{
			1, 1, 0,
			0, 3, 0,
			0, 0, 1,
		}),
		b: []float64{5, 2, 2},
		G: nil,
		h: nil,
		integralityConstraints: []bool{false, true, true},
	}

	//Note:  do not compare pointers
	assert.Equal(t, expected, *solveable)
}

// constraints involving multiple variables and inequalities
func TestProblem_toSolveableE(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable("v1").SetCoeff(-1)
	v2 := prob.AddVariable("v2").SetCoeff(-2).IsInteger()
	v3 := prob.AddVariable("v3").SetCoeff(1).IsInteger()

	// add the equality constraints
	prob.AddConstraint().AddExpression(1, v1).AddExpression(1, v2).EqualTo(5)
	prob.AddConstraint().AddExpression(3, v2).EqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).EqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).AddExpression(1, v1).SmallerThanOrEqualTo(2)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.toSolveable()
	expected := MILPproblem{
		c: []float64{1, 2, -1},
		A: mat.NewDense(3, 3, []float64{
			1, 1, 0,
			0, 3, 0,
			0, 0, 1,
		}),
		b: []float64{5, 2, 2},
		G: mat.NewDense(1, 3, []float64{
			1, 0, 1,
		}),
		h: []float64{2},
		integralityConstraints: []bool{false, true, true},
	}

	//Note:  do not compare pointers
	assert.Equal(t, expected, *solveable)
}

// ONLY inequality constraints
func TestProblem_toSolveableF(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable("v1").SetCoeff(-1)
	v2 := prob.AddVariable("v2").SetCoeff(-2).IsInteger()
	v3 := prob.AddVariable("v3").SetCoeff(1).IsInteger()

	// add the equality constraints
	prob.AddConstraint().AddExpression(1, v1).AddExpression(1, v2).SmallerThanOrEqualTo(5)
	prob.AddConstraint().AddExpression(3, v2).SmallerThanOrEqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).SmallerThanOrEqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).AddExpression(1, v1).SmallerThanOrEqualTo(2)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.toSolveable()
	expected := MILPproblem{
		c: []float64{1, 2, -1},
		A: nil,
		b: nil,
		G: mat.NewDense(4, 3, []float64{
			1, 1, 0,
			0, 3, 0,
			0, 0, 1,
			1, 0, 1,
		}),
		h: []float64{5, 2, 2, 2},
		integralityConstraints: []bool{false, true, true},
	}

	//Note:  do not compare pointers
	assert.Equal(t, expected, *solveable)
}

// With upper and lower bounds on some variables
func TestProblem_toSolveableG(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable("v1").SetCoeff(-1).UpperBound(4).LowerBound(2)
	v2 := prob.AddVariable("v2").SetCoeff(-2).IsInteger()
	v3 := prob.AddVariable("v3").SetCoeff(1).IsInteger().LowerBound(1)

	// add the equality constraints
	prob.AddConstraint().AddExpression(1, v1).AddExpression(1, v2).SmallerThanOrEqualTo(5)
	prob.AddConstraint().AddExpression(3, v2).SmallerThanOrEqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).SmallerThanOrEqualTo(2)
	prob.AddConstraint().AddExpression(1, v3).AddExpression(1, v1).SmallerThanOrEqualTo(2)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.toSolveable()
	expected := MILPproblem{
		c: []float64{1, 2, -1},
		A: nil,
		b: nil,
		G: mat.NewDense(7, 3, []float64{
			1, 1, 0,
			0, 3, 0,
			0, 0, 1,
			1, 0, 1,

			// var bounds
			1, 0, 0,
			-1, 0, 0,
			0, 0, -1,
		}),
		h: []float64{5, 2, 2, 2, 4, -2, -1},
		integralityConstraints: []bool{false, true, true},
	}

	//Note:  do not compare pointers
	assert.Equal(t, expected, *solveable)
}
