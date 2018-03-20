package ilp

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

	solution, err := prob.Solve()
	assert.NoError(t, err)
	assert.Equal(t, float64(-8), solution.solution.z)
	assert.Equal(t, []float64{2, 3, 0, 0}, solution.solution.x)
}

func TestInitialSubproblemSolve(t *testing.T) {
	prob := MILPproblem{
		c: []float64{-1, -2, 0, 0},
		A: mat.NewDense(2, 4, []float64{
			-1, 2, 1, 0,
			3, 1, 0, 1,
		}),
		b: []float64{4, 9},
		integralityConstraints: []bool{false, false, true, false},
	}

	s := prob.toInitialSubProblem()

	solution, err := s.solve()
	t.Log(solution.problem)
	fmt.Println(solution.x)
	assert.NoError(t, err)
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
					c: []float64{-1, -2, 0, 0},
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
				integralityConstraints: []bool{true, false, false, false},
			},
			wantP2: subProblem{
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
				integralityConstraints: []bool{true, false, false, false},
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
				c: []float64{-1, -2, 0, 0},
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
				c: []float64{-1, -2, 0, 0},
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

func TestMILPproblem_Solve(t *testing.T) {
	type fields struct {
		c                      []float64
		A                      *mat.Dense
		b                      []float64
		G                      *mat.Dense
		h                      []float64
		integralityConstraints []bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    MILPsolution
		wantErr bool
	}{
		{
			name: "No integrality constraints, no inequalities",
			fields: fields{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				G: nil,
				h: nil,
				integralityConstraints: []bool{false, false, false, false},
			},
			want: MILPsolution{
				solution: solution{
					x: []float64{2, 3, 0, 0},
					z: float64(-8),
				},
			},
		},
		{
			name: "Intial relaxation satisfies integrality",
			fields: fields{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2, 1, 0,
					3, 1, 0, 1,
				}),
				b: []float64{4, 9},
				G: nil,
				h: nil,
				integralityConstraints: []bool{false, false, false, false},
			},
			want: MILPsolution{
				solution: solution{
					x: []float64{2, 3, 0, 0},
					z: float64(-8),
				},
			},
		},
		{
			name: "No integer feasible solution",
			fields: fields{
				c: []float64{-1, -2, 0, 0},
				A: mat.NewDense(2, 4, []float64{
					-1, 2.6, 1, 0,
					3, 1.1, 0, 1,
				}),
				b: []float64{4, 9},
				G: nil,
				h: nil,
				integralityConstraints: []bool{false, true, false, false},
			},
			want:    MILPsolution{},
			wantErr: true,
		},
		{
			name: "1: One integrality constraint and no initial inequality constraints.",
			fields: fields{
				c: []float64{-1, -2, 0},
				A: mat.NewDense(2, 3, []float64{
					-1, 2.6, 1.2,
					3, 1.1, 1.6,
				}),
				b: []float64{4, 9},
				G: nil,
				h: nil,
				integralityConstraints: []bool{false, false, true},
			},
			want: MILPsolution{
				solution: solution{
					x: []float64{2.134831460674157, 2.3595505617977524, 0},
					z: -6.853932584269662,
				},
			},
			wantErr: false,
		},
		{
			name: "2: One integrality constraint and no initial inequality constraints.",
			fields: fields{
				c: []float64{-1, -2, 1},
				A: mat.NewDense(2, 3, []float64{
					-2, 2.6, 2,
					6, 1.1, 1,
				}),
				b: []float64{4, 9},
				G: nil,
				h: nil,
				integralityConstraints: []bool{false, false, true},
			},
			want: MILPsolution{
				solution: solution{
					x: []float64{1.0674157303370786, 2.3595505617977524, 0},
					z: -5.786516853932583,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := MILPproblem{
				c: tt.fields.c,
				A: tt.fields.A,
				b: tt.fields.b,
				G: tt.fields.G,
				h: tt.fields.h,
				integralityConstraints: tt.fields.integralityConstraints,
			}
			got, err := p.Solve()
			if (err != nil) != tt.wantErr {
				t.Log(got.decisionLog)
				t.Errorf("MILPproblem.Solve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Note: we compare only the numerical solution variables
			if !(reflect.DeepEqual(tt.want.solution.x, got.solution.x) && tt.want.solution.z == got.solution.z) {
				t.Log(got.decisionLog)
				t.Errorf("MILPproblem.Solve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandomized(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))

	// some small problems
	testRandomMILP(t, 100, 0, 10, rnd)

	// some small problems with some zeros
	testRandomMILP(t, 100, 0.1, 10, rnd)

	// larger problems
	testRandomMILP(t, 100, 0, 100, rnd)
}

func testRandomMILP(t *testing.T, nTest int, pZero float64, maxN int, rnd *rand.Rand) {
	// Try a bunch of random LPs
	for i := 0; i < nTest; i++ {
		n := rnd.Intn(maxN) + 2 // n must be at least two.
		m := rnd.Intn(n-1) + 1  // m must be between 1 and n
		prob := getRandomMILP(pZero, m, n, rnd)

		fmt.Println("c:")
		fmt.Println(prob.c)
		fmt.Println("A:")
		fmt.Println(mat.Formatted(prob.A))
		fmt.Println("b:")
		fmt.Println(prob.b)
		solution, err := prob.Solve()

		fmt.Println(solution.solution.x, solution.solution.z, err)
	}
}

// adapted from Gonum's lp.Simplex.
func getRandomMILP(pZero float64, m, n int, rnd *rand.Rand) *MILPproblem {

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
	a := mat.NewDense(m, n, nil)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			a.Set(i, j, randValue())
		}
	}

	b := make([]float64, m)
	for i := range b {
		b[i] = randValue()
	}

	c := make([]float64, n)
	for i := range c {
		c[i] = randValue()
	}

	g := mat.NewDense(m, n, nil)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			g.Set(i, j, randValue())
		}
	}

	h := make([]float64, m)
	for i := range h {
		h[i] = randValue()
	}

	boolgenerator := NewBoolGen(rnd)

	var integralityConstraints []bool
	for i := 0; i < len(c); i++ {
		integralityConstraints = append(integralityConstraints, boolgenerator.Bool())
	}
	if len(c) != len(integralityConstraints) {
		panic("randomized constraint vector and c vector not of equal length")
	}
	return &MILPproblem{
		c: c,
		A: a,
		b: b,
		G: g,
		h: h,
		integralityConstraints: integralityConstraints,
	}
}

// random boolean generator
type Boolgen struct {
	src       rand.Source
	cache     int64
	remaining int
}

func NewBoolGen(rnd rand.Source) *Boolgen {
	return &Boolgen{src: rnd}
}

func (b *Boolgen) Bool() bool {
	if b.remaining == 0 {
		b.cache, b.remaining = b.src.Int63(), 63
	}

	result := b.cache&0x01 == 1
	b.cache >>= 1
	b.remaining--

	return result
}
