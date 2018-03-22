package ilp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeasibleForIP(t *testing.T) {

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
		assert.Equal(t, testd.shouldPass, feasibleForIP(testd.constraints, testd.solution))
	}
}
