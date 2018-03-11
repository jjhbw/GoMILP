package ilp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/mat"
)

func TestExampleSimplex(t *testing.T) {
	ExampleSimplex()
}

func TestMILPproblem_Solve_NoInteger(t *testing.T) {
	prob := MILPproblem{
		c: []float64{-1, -2, 0, 0},
		A: mat.NewDense(2, 4, []float64{
			-1, 2, 1, 0,
			3, 1, 0, 1,
		}),
		b:                []float64{4, 9},
		integerVariables: []bool{false, false, false, false},
	}

	z, x, err := prob.Solve()
	assert.NoError(t, err)
	assert.Equal(t, float64(-8), z)
	assert.Equal(t, x, []float64{2, 3, 0, 0})
}
