package ilp

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
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
