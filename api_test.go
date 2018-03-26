package ilp

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/mat"
)

func TestProblem_checkExpression(t *testing.T) {

	// a true case
	prob := NewProblem()
	v := prob.AddVariable("v1").SetCoeff(1)

	expr1 := expression{
		variable: v,
		coef:     2,
	}
	assert.True(t, prob.checkExpression(expr1))

	// an expression with a new variable not declared in the problem
	expr2 := expression{
		variable: &Variable{coefficient: 1, integer: false},
		coef:     1,
	}
	assert.False(t, prob.checkExpression(expr2))

}

// adapted from Gonum's lp.Simplex.
func getRandomProblem(pZero float64, m, n int, rnd *rand.Rand) Problem {

	if m == 0 || n == 0 {
		panic("m==n not allowed")
	}
	randValue := func() float64 {
		//var pZero float64
		v := rnd.Float64()
		if v < pZero {
			return 0
		}
		return rnd.NormFloat64()
	}

	boolgenerator := NewBoolGen(rnd)
	prob := NewProblem()

	var vars []*Variable

	// add variables
	for i := 0; i < m; i++ {
		v := prob.AddVariable(fmt.Sprintf("%v", i)).SetCoeff(randValue())
		if boolgenerator.Bool() {
			v.IsInteger()
		}
		vars = append(vars, v)
	}

	for _, v := range vars {
		// add (at least) one constraint for each variable

		// TODO: more complex constraint matrices
		// for j := 0; j < m; j++ {
		// 	if boolgenerator.Bool() && boolgenerator.Bool() {
		// 		exprs = append(exprs, expression{randValue(), v})
		// 	}
		// }

		// roll the dice on whether it will become an equality or inequality
		con := prob.AddConstraint().AddExpression(randValue(), v)
		if boolgenerator.Bool() {
			con.EqualTo(randValue())
		} else {
			con.SmallerThanOrEqualTo(randValue())
		}

	}

	return prob
}

func TestProblem_Solve(t *testing.T) {

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
	expected := milpProblem{
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

	// check that the conversion was successful
	// Note:  do not compare pointers
	assert.Equal(t, expected, *solveable)

	// solve the problem directly (without any timeouts)
	soln, err := prob.Solve(context.Background())
	assert.NoError(t, err)

	getVal := func(n string) float64 {
		x, err := soln.GetValueFor(n)
		assert.NoError(t, err)
		return x
	}

	// check whether the found coefficient values are as expected
	assert.Equal(t, getVal("v1"), float64(5))
	assert.Equal(t, getVal("v2"), float64(0.6666666666666666))
	assert.Equal(t, getVal("v3"), float64(2))
	assert.Equal(t, getVal("v4"), float64(0))

}
