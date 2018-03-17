package ilp

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProblem_checkExpression(t *testing.T) {

	// a true case
	prob := NewProblem()
	v := prob.AddVariable(1, false)

	expr1 := Expression{
		variable: v,
		coef:     2,
	}
	assert.True(t, prob.checkExpression(expr1))

	// an expression with a new variable not declared in the problem
	expr2 := Expression{
		variable: &Variable{Coefficient: 1, Integer: false},
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
		v := prob.AddVariable(randValue(), boolgenerator.Bool())
		vars = append(vars, v)
	}

	for _, v := range vars {
		// add (at least) one constraint for each variable
		exprs := []Expression{Expression{randValue(), v}}

		// TODO: more complex constraint matrices
		// for j := 0; j < m; j++ {
		// 	if boolgenerator.Bool() && boolgenerator.Bool() {
		// 		exprs = append(exprs, Expression{randValue(), v})
		// 	}
		// }

		// roll the dice on whether it will become an equality or inequality
		if boolgenerator.Bool() {
			prob.AddEquality(exprs, randValue())
		} else {
			prob.AddInEquality(exprs, randValue())
		}

	}

	return prob
}
