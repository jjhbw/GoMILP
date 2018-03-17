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
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, false)
	v3 := prob.AddVariable(1, false)
	v4 := prob.AddVariable(3, false)

	// add the equality constraints
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v1,
		},
	},
		5,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     3,
			variable: v2,
		},
	},
		2,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v3,
		},
	},
		2,
	)

	// add the inequality
	prob.AddInEquality([]Expression{
		Expression{
			coef:     1,
			variable: v4,
		},
	},
		2,
	)

	solveable := prob.ToSolveable()
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
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, true)
	v3 := prob.AddVariable(1, true)

	// add the equality constraints
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v1,
		},
	},
		5,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     3,
			variable: v2,
		},
	},
		2,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v3,
		},
	},
		2,
	)

	solveable := prob.ToSolveable()
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
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, true)
	v3 := prob.AddVariable(1, true)

	// add the equality constraints
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v1,
		},
	},
		5,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     3,
			variable: v2,
		},
	},
		2,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v3,
		},
	},
		2,
	)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.ToSolveable()
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
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, true)
	v3 := prob.AddVariable(1, true)

	// add the equality constraints
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v1,
		},
		Expression{
			coef:     1,
			variable: v2,
		},
	},
		5,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     3,
			variable: v2,
		},
	},
		2,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v3,
		},
	},
		2,
	)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.ToSolveable()
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
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, true)
	v3 := prob.AddVariable(1, true)

	// add the equality constraints
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v1,
		},
		Expression{
			coef:     1,
			variable: v2,
		},
	},
		5,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     3,
			variable: v2,
		},
	},
		2,
	)
	prob.AddEquality([]Expression{
		Expression{
			coef:     1,
			variable: v3,
		},
	},
		2,
	)
	prob.AddInEquality([]Expression{
		Expression{
			coef:     1,
			variable: v3,
		},
		Expression{
			coef:     1,
			variable: v1,
		},
	},
		2,
	)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.ToSolveable()
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
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, true)
	v3 := prob.AddVariable(1, true)

	// add the equality constraints
	prob.AddInEquality([]Expression{
		Expression{
			coef:     1,
			variable: v1,
		},
		Expression{
			coef:     1,
			variable: v2,
		},
	},
		5,
	)
	prob.AddInEquality([]Expression{
		Expression{
			coef:     3,
			variable: v2,
		},
	},
		2,
	)
	prob.AddInEquality([]Expression{
		Expression{
			coef:     1,
			variable: v3,
		},
	},
		2,
	)
	prob.AddInEquality([]Expression{
		Expression{
			coef:     1,
			variable: v3,
		},
		Expression{
			coef:     1,
			variable: v1,
		},
	},
		2,
	)

	// set the problem to maximize
	prob.Maximize()

	solveable := prob.ToSolveable()
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
