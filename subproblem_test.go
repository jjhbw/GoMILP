package ilp

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize/convex/lp"
)

func TestExampleSimplex(t *testing.T) {
	// standard form:
	// 	minimize	c^T x
	// s.t. 		A * x = b
	// 				x >= 0 .

	// this example solves the following problem:
	// Minimize Z = -1x1 + -2x2 + 0x3 + 0x4
	// Subject to:
	//		-1x1 	+ 2x2 	+ 1x3 	+ 0x4 	= 4
	//		3x1 	+ 1x2 	+ 0x3 	+ 1x4 	= 9

	c := []float64{-1, -2, 0, 0}
	A := mat.NewDense(2, 4, []float64{
		-1, 2, 1, 0,
		3, 1, 0, 1,
	})
	b := []float64{4, 9}

	z, x, err := lp.Simplex(c, A, b, 0, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("opt: %v\n", z)
	fmt.Printf("x: %v\n", x)
	// Output:
	// z: -8
	// x: [2 3 0 0]
}

func Test_subProblem_combineInequalities(t *testing.T) {
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
			name: "no bnb constraints",
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
			name: "One bnb constraint",
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
			name: "Two bnb constraints",
			fields: fields{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				bnbConstraints: []bnbConstraint{
					{
						branchedVariable: 3,
						hsharp:           1,
						gsharp:           []float64{0, 0, 0, 1},
					},
					{
						branchedVariable: 1,
						hsharp:           3,
						gsharp:           []float64{0, 1, 0, 0},
					},
				},
			},
			want: mat.NewDense(2, 4, []float64{
				0, 0, 0, 1,
				0, 1, 0, 0,
			}),
			want1: []float64{1, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := subProblem{
				c:              tt.fields.c,
				A:              tt.fields.A,
				b:              tt.fields.b,
				bnbConstraints: tt.fields.bnbConstraints,
			}
			got, got1 := p.combineInequalities()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("subProblem.getInequalities() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("subProblem.getInequalities() got1 = %v, want %v", got1, tt.want1)
			}
		})
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
		wantP1 subProblem
		wantP2 subProblem
	}{
		{
			name: "branch on first variable",
			fields: fields{
				problem: &subProblem{
					id:     0,
					parent: 0,
					c:      []float64{-1, -2, 0, 0},
					A: mat.NewDense(2, 4, []float64{
						-1, 2, 1, 0,
						3, 1, 0, 1,
					}),
					b: []float64{4, 9},
					integralityConstraints: []bool{true, false, false, false},
				},
				// a fake problem. This solution does not have to be true or feasible.
				x: []float64{1.2, 3, 0, 0},
				z: float64(-8),
			},
			wantP1: subProblem{
				id:     0,
				parent: 0,
				c:      []float64{-1, -2, 0, 0},
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
				integralityConstraints: []bool{true, false, false, false},
			},
			wantP2: subProblem{
				id:     0,
				parent: 0,
				c:      []float64{-1, -2, 0, 0},
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
				integralityConstraints: []bool{true, false, false, false},
			},
		},
		{
			name: "branch on second variable",
			fields: fields{
				problem: &subProblem{
					id: 1,
					c:  []float64{-1, -2, 0, 0},
					A: mat.NewDense(2, 4, []float64{
						-1, 2, 1, 0,
						3, 1, 0, 1,
					}),
					b: []float64{4, 9},
					integralityConstraints: []bool{true, true, false, false},
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
			wantP1: subProblem{
				id:     0,
				parent: 1,
				c:      []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				integralityConstraints: []bool{true, true, false, false},
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
			wantP2: subProblem{
				id:     0,
				parent: 1,
				c:      []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				integralityConstraints: []bool{true, true, false, false},
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
			gotP1, gotP2 := s.branch()
			if !reflect.DeepEqual(gotP1, tt.wantP1) {
				t.Errorf("solution.branch() gotP1 = %v, want %v", gotP1, tt.wantP1)
			}
			if !reflect.DeepEqual(gotP2, tt.wantP2) {
				t.Errorf("solution.branch() gotP2 = %v, want %v", gotP2, tt.wantP2)
			}
		})
	}
}

func Test_convertToEqualities(t *testing.T) {
	type args struct {
		c []float64
		A *mat.Dense
		b []float64
		G *mat.Dense
		h []float64
	}
	tests := []struct {
		name     string
		args     args
		wantCNew []float64
		wantANew *mat.Dense
		wantBNew []float64
	}{
		{
			name: "simple case",
			args: args{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				h: []float64{2, 5, 8},
				G: mat.NewDense(3, 4, []float64{
					0, 0, 0, 1,
					0, 0, 1, 0,
					0, 1, 0, 0}),
			},
			wantCNew: []float64{-1, -2, 0, 0, 0, 0, 0},
			wantANew: mat.NewDense(5, 7, []float64{
				-1, 2, 1, 0, 0, 0, 0,
				3, 1, 0, 1, 0, 0, 0,
				0, 0, 0, 1, 1, 0, 0,
				0, 0, 1, 0, 0, 1, 0,
				0, 1, 0, 0, 0, 0, 1}),
			wantBNew: []float64{4, 9, 2, 5, 8},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCNew, gotANew, gotBNew := convertToEqualities(tt.args.c, tt.args.A, tt.args.b, tt.args.G, tt.args.h)
			fmt.Println("Cnew:")
			fmt.Println(gotCNew)
			fmt.Println("ANew:")
			fmt.Println(mat.Formatted(gotANew))
			fmt.Println("BNew:")
			fmt.Println(gotBNew)

			if !reflect.DeepEqual(gotCNew, tt.wantCNew) {
				t.Errorf("addSlackVariables() gotCNew = %v, want %v", gotCNew, tt.wantCNew)
			}
			if !reflect.DeepEqual(gotANew, tt.wantANew) {
				t.Errorf("addSlackVariables() gotANew = %v, want %v", gotANew, tt.wantANew)
			}
			if !reflect.DeepEqual(gotBNew, tt.wantBNew) {
				t.Errorf("addSlackVariables() gotBNew = %v, want %v", gotBNew, tt.wantBNew)
			}
		})
	}
}
