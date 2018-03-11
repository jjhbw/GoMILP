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

func TestIntegralityConstraintsCheck(t *testing.T) {

	testdata := []struct {
		constraints []bool
		solution    []float64
		shouldPass  bool
	}{
		{
			constraints: []bool{false, false, false, false},
			solution:    []float64{1, 2, 3, 4.5},
			shouldPass:  true,
		},
		{
			constraints: []bool{false, false, false, true},
			solution:    []float64{1, 2, 3, 4.5},
			shouldPass:  false,
		},
		{
			constraints: []bool{true, false, false, true},
			solution:    []float64{1, 2, 3, 4.5},
			shouldPass:  false,
		},
		{
			constraints: []bool{true, true, true, true},
			solution:    []float64{1, 2, 3, 4},
			shouldPass:  true,
		},
	}

	for _, testd := range testdata {
		assert.Equal(t, testd.shouldPass, satisfiesIntegralityConstraints(testd.constraints, testd.solution))
	}
}
