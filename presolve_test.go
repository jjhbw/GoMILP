package ilp

import (
	"reflect"
	"testing"
)

func Test_preProcessor_filterFixedVars(t *testing.T) {

	tests := []struct {
		name          string
		getRawProblem func() (Problem, []*Variable)
		want          Problem
	}{
		{
			name: "one fixed var",
			getRawProblem: func() (Problem, []*Variable) {
				// return a new problem and the variables that we want removed
				prob := NewProblem()
				okayvar := prob.AddVariable("okayvar").LowerBound(1).UpperBound(3)

				okayVars := []*Variable{
					okayvar,
				}

				// the offender
				prob.AddVariable("notokayvar").LowerBound(1).UpperBound(1)

				return prob, okayVars
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob, okayvars := tt.getRawProblem()
			prepper := newPreprocessor()

			preppedProb := prepper.filterFixedVars(prob)

			// check the variables
			if !reflect.DeepEqual(preppedProb.variables, okayvars) {
				t.Errorf("Unexpected set of variables returned by fixed variable filter! got: %v, want: %v", preppedProb.variables, okayvars)
			}

			// check if the removed variables are not accidentally still referenced in the constraints
			for _, c := range preppedProb.constraints {
				for _, e := range c.expressions {
					okay := false
				check:
					for _, v := range okayvars {
						if e.variable == v {
							okay = true
							break check
						}
					}
					if !okay {
						t.Errorf("Variable %v is still present in the constraint expressions!", e)
					}
				}
			}

		})
	}
}
