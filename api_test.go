package ilp

import (
	"fmt"
	"testing"

	"gonum.org/v1/gonum/mat"

	"github.com/lukpank/go-glpk/glpk"
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

// Convert the problem to a GLPK problem using its terrible API
func ToGLPK(p Problem) *glpk.Prob {
	converted := glpk.New()

	converted.SetProbName("sample")
	converted.SetObjName("Z")

	if p.maximize {
		converted.SetObjDir(glpk.MAX)
	} else {
		converted.SetObjDir(glpk.MIN)
	}

	// define the problem dimensions
	converted.AddRows(len(p.equalities) + len(p.inequalities))
	converted.AddCols(len(p.variables))

	// add the variables
	for i := 0; i < len(p.variables); i++ {
		name := fmt.Sprintf("x%d", i)
		colInd := i + 1
		converted.SetColName(colInd, name)

		// set the objective coeff
		converted.SetObjCoef(colInd, p.variables[i].Coefficient)

		// give all variables a lower bound of 0
		converted.SetColBnds(colInd, glpk.LO, 0.0, 0.0)

		// set integrality constraint, if any
		if p.variables[i].Integer {
			converted.SetColKind(colInd, glpk.IV)
		}
	}

	// // add the equality constraints
	for i, equality := range p.equalities {

		// build the matrix row for the equality
		equalityCoefs := []float64{0} // add a zero, see details on this weird glpk api nuance below
		indices := []int32{0}
		for _, exp := range equality.expressions {
			for i, va := range p.variables {
				if exp.variable == va {
					indices = append(indices, int32(i)+1)
					equalityCoefs = append(equalityCoefs, exp.coef)
				}
			}
		}

		eqRow := converted.AddRows(1)                              // returns the index of the added row
		converted.SetRowName(eqRow, fmt.Sprintf("equality_%v", i)) // name the row for debugging purposes
		converted.SetMatRow(eqRow, indices, equalityCoefs)         // NOTE: from the docs: "ind[0] and val[0] are ignored", so a leading 0 is given in both vectors."
		converted.SetRowBnds(eqRow, glpk.FX, equality.equalTo, 0)
	}

	// // add the inequality constraints
	for i, ineq := range p.inequalities {

		// build the matrix row for the equality
		inEqualityCoefs := []float64{0} // add a zero, see details on this weird glpk api nuance below
		indices := []int32{0}

		for _, exp := range ineq.expressions {
			for i, va := range p.variables {
				if exp.variable == va {
					indices = append(indices, int32(i)+1)
					inEqualityCoefs = append(inEqualityCoefs, exp.coef)
				}
			}
		}
		ineqRow := converted.AddRows(1)                                // returns the index of the added row
		converted.SetRowName(ineqRow, fmt.Sprintf("inequality_%v", i)) // name the row for debugging purposes
		converted.SetMatRow(ineqRow, indices, inEqualityCoefs)         // NOTE: from the docs: "ind[0] and val[0] are ignored", so a leading 0 is given in both vectors."
		converted.SetRowBnds(ineqRow, glpk.FX, ineq.smallerThan, 0)
	}

	return converted
}

func TestCompareWithGLPK(t *testing.T) {
	// build an abstract Problem
	prob := NewProblem()

	// add the variables
	v1 := prob.AddVariable(-1, false)
	v2 := prob.AddVariable(-2, false)
	v3 := prob.AddVariable(0, true)

	// add the equality constraints
	prob.AddEquality([]Expression{
		Expression{coef: -1, variable: v1},
		Expression{coef: 2.6, variable: v2},
		Expression{coef: 1.2, variable: v3},
	}, 4)
	prob.AddEquality([]Expression{
		Expression{coef: 3, variable: v1},
		Expression{coef: 1.1, variable: v2},
		Expression{coef: 1.6, variable: v3},
	}, 9)

	// set the problem to maximize
	prob.Minimize()

	// solve the problem using our own code
	solution, err := prob.toSolveable().Solve()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(solution.decisionLog)
	fmt.Println(solution.solution)

	// convert the problem to glpk
	glpkProblem := ToGLPK(prob)

	// save the problem for debugging purposes
	glpkProblem.WriteLP(nil, "test.lp")

	// solve the problem with the integer solver
	iocp := glpk.NewIocp()
	iocp.SetPresolve(true)
	solveError := glpkProblem.Intopt(iocp)
	if solveError != nil {
		t.Error(solveError)
	}

	// parse the solutions
	fmt.Printf("%s = %g", glpkProblem.ObjName(), glpkProblem.ObjVal())
	for i := 0; i < 3; i++ {
		fmt.Printf("; %s = %g", glpkProblem.ColName(i+1), glpkProblem.MipColVal(i+1))
	}
	fmt.Println()

	t.Fail()
}
