package ilp

import (
	"testing"

	"gonum.org/v1/gonum/mat"

	"github.com/stretchr/testify/assert"
)

func TestProblem_checkExpression(t *testing.T) {

	// a true case
	prob := NewProblem()
	v := prob.AddVariable(1, false)

	expr1 := expression{
		variable: v,
		coef:     2,
	}
	assert.True(t, prob.checkExpression(expr1))

	// an expression with a new variable not declared in the problem
	expr2 := expression{
		variable: &Variable{Coefficient: 1, Integer: false},
		coef:     1,
	}
	assert.False(t, prob.checkExpression(expr2))

}

// a simple case with one inequality and no integrality constraints
func TestProblem_toSolveableA(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, false)
	v3 := prob.AddVariable(1, false)
	v4 := prob.AddVariable(3, false)

	// add the equality constraints
	prob.AddEquality([]expression{
		expression{
			coef:     1,
			variable: v1,
		},
	},
		5,
	)
	prob.AddEquality([]expression{
		expression{
			coef:     3,
			variable: v2,
		},
	},
		2,
	)
	prob.AddEquality([]expression{
		expression{
			coef:     1,
			variable: v3,
		},
	},
		2,
	)

	// add the inequality
	prob.AddInEquality([]expression{
		expression{
			coef:     1,
			variable: v4,
		},
	},
		2,
	)

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

// No inequalities and 2 integrality constraints
func TestProblem_toSolveableB(t *testing.T) {

	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, true)
	v3 := prob.AddVariable(1, true)

	// add the equality constraints
	prob.AddEquality([]expression{
		expression{
			coef:     1,
			variable: v1,
		},
	},
		5,
	)
	prob.AddEquality([]expression{
		expression{
			coef:     3,
			variable: v2,
		},
	},
		2,
	)
	prob.AddEquality([]expression{
		expression{
			coef:     1,
			variable: v3,
		},
	},
		2,
	)

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
