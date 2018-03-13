package ilp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/mat"
)

func TestExampleSimplex(t *testing.T) {
	ExampleSimplex()
}

func TestMILPproblem_Solve_Smoke_NoInteger(t *testing.T) {
	prob := MILPproblem{
		c: []float64{-1, -2, 0, 0},
		A: mat.NewDense(2, 4, []float64{
			-1, 2, 1, 0,
			3, 1, 0, 1,
		}),
		b: []float64{4, 9},
		integralityConstraints: []bool{false, false, false, false},
	}

	z, x, err := prob.Solve()
	assert.NoError(t, err)
	assert.Equal(t, float64(-8), z)
	assert.Equal(t, x, []float64{2, 3, 0, 0})
}

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

func Test_solution_branch(t *testing.T) {
	type fields struct {
		problem *subProblem
		x       []float64
		z       float64
	}

	tests := []struct {
		name   string
		fields fields
		want1  subProblem
		want2  subProblem
	}{
		{
			name: "branch on first variable",
			fields: fields{
				problem: &subProblem{
					c: []float64{-1, -2, 0, 0},
					A: mat.NewDense(2, 4, []float64{
						-1, 2, 1, 0,
						3, 1, 0, 1,
					}),
					b: []float64{4, 9},
				},
				// a fake problem. This solution does not have to be true or feasible.
				x: []float64{1.2, 3, 0, 0},
				z: float64(-8),
			},
			want1: subProblem{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				bnbConstraints: []bnbConstraint{
					{
						branchedVariable: 0,
						hsharp:           1,
						gsharp:           []float64{1, 0, 0, 0},
					},
				},
			},
			want2: subProblem{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				bnbConstraints: []bnbConstraint{
					{
						branchedVariable: 0,
						hsharp:           -2,
						gsharp:           []float64{-1, 0, 0, 0},
					},
				},
			},
		},
		{
			name: "branch on second variable",
			fields: fields{
				problem: &subProblem{
					c: []float64{-1, -2, 0, 0},
					A: mat.NewDense(2, 4, []float64{
						-1, 2, 1, 0,
						3, 1, 0, 1,
					}),
					b: []float64{4, 9},
					bnbConstraints: []bnbConstraint{
						{
							branchedVariable: 0,
							hsharp:           1,
							gsharp:           []float64{1, 0, 0, 0},
						},
					},
				},
				// a fake problem. This solution does not have to be true or feasible.
				x: []float64{1.2, 3.8, 0, 0},
				z: float64(-8),
			},
			want1: subProblem{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				bnbConstraints: []bnbConstraint{
					{
						branchedVariable: 0,
						hsharp:           1,
						gsharp:           []float64{1, 0, 0, 0},
					},
					{
						branchedVariable: 1,
						hsharp:           3,
						gsharp:           []float64{0, 1, 0, 0},
					},
				},
			},
			want2: subProblem{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				bnbConstraints: []bnbConstraint{
					{
						branchedVariable: 0,
						hsharp:           1,
						gsharp:           []float64{1, 0, 0, 0},
					},
					{
						branchedVariable: 1,
						hsharp:           -4,
						gsharp:           []float64{0, -1, 0, 0},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := solution{
				problem: tt.fields.problem,
				x:       tt.fields.x,
				z:       tt.fields.z,
			}
			got1, got2 := s.branch()
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("solution.branch() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("solution.branch() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_subProblem_getInequalities(t *testing.T) {
	type fields struct {
		c              []float64
		A              *mat.Dense
		b              []float64
		G              *mat.Dense
		h              []float64
		bnbConstraints []bnbConstraint
	}
	tests := []struct {
		name   string
		fields fields
		want   *mat.Dense
		want1  []float64
	}{
		{
			name: "no bnb or original constraints",
			fields: fields{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
			},
			want:  nil,
			want1: nil,
		},
		{
			name: "only original constraints",
			fields: fields{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				h: []float64{1},
				G: mat.NewDense(1, 4, []float64{1, 0, 0, 0}),
			},
			want:  mat.NewDense(1, 4, []float64{1, 0, 0, 0}),
			want1: []float64{1},
		},
		{
			name: "One bnb constraint, no original inequality constraints",
			fields: fields{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				bnbConstraints: []bnbConstraint{
					{
						branchedVariable: 0,
						hsharp:           1,
						gsharp:           []float64{1, 0, 0, 0},
					},
				},
			},
			want:  mat.NewDense(1, 4, []float64{1, 0, 0, 0}),
			want1: []float64{1},
		},
		{
			name: "One bnb constraint, one original inequality constraint",
			fields: fields{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				h: []float64{2},
				G: mat.NewDense(1, 4, []float64{0, 0, 0, 1}),
				bnbConstraints: []bnbConstraint{
					{
						branchedVariable: 0,
						hsharp:           1,
						gsharp:           []float64{1, 0, 0, 0},
					},
				},
			},
			want:  mat.NewDense(2, 4, []float64{0, 0, 0, 1, 1, 0, 0, 0}),
			want1: []float64{2, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := subProblem{
				c:              tt.fields.c,
				A:              tt.fields.A,
				b:              tt.fields.b,
				G:              tt.fields.G,
				h:              tt.fields.h,
				bnbConstraints: tt.fields.bnbConstraints,
			}
			got, got1 := p.getInequalities()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("subProblem.getInequalities() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("subProblem.getInequalities() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
