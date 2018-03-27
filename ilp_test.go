package ilp

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/mat"
)

func TestmilpProblem_Solve_Smoke_NoInteger(t *testing.T) {
	prob := milpProblem{
		c: []float64{-1, -2, 0, 0},
		A: mat.NewDense(2, 4, []float64{
			-1, 2, 1, 0,
			3, 1, 0, 1,
		}),
		b: []float64{4, 9},
		integralityConstraints: []bool{false, false, false, false},
	}

	// solve the problem with 1 worker and a one-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := prob.solve(ctx, 1, dummyMiddleware{})

	assert.NoError(t, err)
	assert.Equal(t, float64(-8), got.solution.z)
	assert.Equal(t, []float64{2, 3, 0, 0}, got.solution.x)
}

func TestInitialSubproblemSolve(t *testing.T) {
	prob := milpProblem{
		c: []float64{-1, -2, 0, 0},
		A: mat.NewDense(2, 4, []float64{
			-1, 2, 1, 0,
			3, 1, 0, 1,
		}),
		b: []float64{4, 9},
		integralityConstraints: []bool{false, false, true, false},
	}

	s := prob.toInitialSubProblem()

	solution := s.solve()
	t.Log(solution.problem)
	fmt.Println(solution.x)
	assert.NoError(t, solution.err)
}

// A regression test case for a potential infinite recursion in the branch-and-bound procedure.
func TestMilpProblem_Solve_InfiniteRecursion_Regression(t *testing.T) {

	prob := milpProblem{
		c: []float64{1.7356332566545616, -0.2058339272568599, -1.051665297603944},
		A: mat.NewDense(1, 3, []float64{
			-0.7762132098737671, 1.42027949678888, -0.3304567624749696,
		}),
		b: []float64{-0.24703471683023603},
		G: mat.NewDense(1, 3, []float64{
			-0.6775235462631393, -1.9616379110849085, 1.9859192819811322,
		}),
		h: []float64{-0.041138108068992485},
		integralityConstraints: []bool{true, true, true},
	}

	want := milpSolution{}

	// solve the problem with 2 workers and a one-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := prob.solve(ctx, 2, dummyMiddleware{})

	assert.Error(t, err)
	assert.Equal(t, err, context.DeadlineExceeded)

	if !(reflect.DeepEqual(want.solution.x, got.solution.x) && want.solution.z == got.solution.z) {
		t.Log(got)
		t.Errorf("milpProblem.SolveWithCtx() = %v, want %v", got, want)
	}

}

// a regression test for nil returns of the search procedure that caused a panic
func TestMilpProblem_Solve_NilReturn_Regression(t *testing.T) {

	prob := milpProblem{
		c: []float64{0.6572445982216386, -1.2787102180406373, -0.714364219639056, 0.4294876505980715, -1.2694040908754067},
		A: mat.NewDense(3, 5, []float64{
			-1.150658083043829, 0.6742357592398329, 0.05482227950158375, -0.4402215293563758, -0.42514963905670267,
			1.8805693836928625, 1.2321077204169477, -1.4072763551877006, 0.32105052839669324, 0.8175654516598202,
			-1.2427589013990952, 0.8480328391203368, 1.8893229216030778, 1.6284926471665957, -0.6924382873998646,
		}),
		b: []float64{-1.6441336258376302, 1.7731638122722604, 0.41457840377809935},
		G: mat.NewDense(3, 5, []float64{
			0.5833490684770126, -0.7706968790319841, 0.6630978893449531, -0.560670828793711, -0.9502215220573013,
			-0.25962903857408626, -0.613464243927484, 0.8559661237279594, -2.5511417937898293, 0.8262232497486882,
			-1.136768995071479, -0.5756455306742008, -1.372457014240165, 0.21778519481503805, 2.7692491194887667,
		}),
		h: []float64{0.12870156802034122, -0.3689382882114889, 0.1658000515068819},
		integralityConstraints: []bool{true, false, false, true, false},
	}

	want := milpSolution{}

	// solve the problem with 2 workers and a one-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := prob.solve(ctx, 2, dummyMiddleware{})

	assert.Error(t, err)
	assert.Equal(t, err, NO_INTEGER_FEASIBLE_SOLUTION)

	if !(reflect.DeepEqual(want.solution.x, got.solution.x) && want.solution.z == got.solution.z) {
		t.Log(got)
		t.Errorf("milpProblem.SolveWithCtx() = %v, want %v", got, want)
	}

}

func TestMilpProblem_SolveMultiple(t *testing.T) {
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
		want    milpSolution
		wantErr error
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
			want: milpSolution{
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
			want: milpSolution{
				solution: solution{
					x: []float64{2, 3, 0, 0},
					z: float64(-8),
				},
			},
		},
		{
			name: "1: One integrality constraint and no initial inequality constraints.",
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
			want: milpSolution{
				solution: solution{
					x: []float64{2.2666666666666666, 2, 1.0666666666666664, 0},
					z: -6.266666666666667,
				},
			},
		},
		{
			name: "2: One integrality constraint and no initial inequality constraints.",
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
			want: milpSolution{
				solution: solution{
					x: []float64{2.134831460674157, 2.3595505617977524, 0},
					z: -6.853932584269662,
				},
			},
		},
		{
			name: "3: One integrality constraint and no initial inequality constraints.",
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
			want: milpSolution{
				solution: solution{
					x: []float64{1.0674157303370786, 2.3595505617977524, 0},
					z: -5.786516853932583,
				},
			},
		},
		{
			name: "One integrality constraint and one initial inequality constraint.",
			fields: fields{
				c: []float64{-1, -2, 1},
				A: mat.NewDense(2, 3, []float64{
					-2, 2.6, 2,
					6, 1.1, 1,
				}),
				b: []float64{4, 9},
				G: mat.NewDense(1, 3, []float64{
					-1, 0, 0,
				}),
				h: []float64{-1},
				integralityConstraints: []bool{false, false, true},
			},
			want: milpSolution{
				solution: solution{
					x: []float64{1.0674157303370786, 2.359550561797753, 0},
					z: -5.786516853932584,
				},
			},
		},
		{
			// regression case that led to a race condition due in-place modification of subProblem child constraints
			// and to infinite recursion in the branch-and-bound procedure
			name: "infinite recursion regression: two integrality constraints and two initial inequality constraints.",
			fields: fields{
				c: []float64{1.7356332566545616, -0.2058339272568599, -1.051665297603944},
				A: mat.NewDense(1, 3, []float64{
					-0.7762132098737671, 1.42027949678888, -0.3304567624749696,
				}),
				b: []float64{-0.24703471683023603},
				G: mat.NewDense(1, 3, []float64{
					-0.6775235462631393, -1.9616379110849085, 1.9859192819811322,
				}),
				h: []float64{-0.041138108068992485},
				integralityConstraints: []bool{true, true, true},
			},
			want:    milpSolution{},
			wantErr: context.DeadlineExceeded,
		},
	}
	for _, tt := range tests {

		// Run the tests with a varying number of workers
		for i := 1; i <= 4; i++ {

			testname := fmt.Sprintf("%v | workers: %v", tt.name, i)

			t.Run(testname, func(t *testing.T) {
				p := milpProblem{
					c: tt.fields.c,
					A: tt.fields.A,
					b: tt.fields.b,
					G: tt.fields.G,
					h: tt.fields.h,
					integralityConstraints: tt.fields.integralityConstraints,
				}

				// solve the problem with 'i' workers and a one-second timeout
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				got, err := p.solve(ctx, i, dummyMiddleware{})
				if err != tt.wantErr {
					t.Log(got)
					t.Errorf("milpProblem.SolveWithCtx() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				// Note: we compare only the numerical solution variables
				if !(reflect.DeepEqual(tt.want.solution.x, got.solution.x) && tt.want.solution.z == got.solution.z) {
					t.Log(got)
					t.Errorf("milpProblem.SolveWithCtx() = %v, want %v %v", got, tt.want.solution.x, tt.want.solution.z)
				}
			})
		}

	}
}

// Test a series of randomly generated problems, hunting for panics.
func TestRandomized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping randomized testing in short mode!")
	}

	rnd := rand.New(rand.NewSource(1))

	workerRange := 3
	for i := 1; i <= workerRange; i++ {

		// some small problems
		testRandomMILP(t, 100, 0, 10, rnd, i)

		// larger problems with some zeros
		testRandomMILP(t, 100, 0.1, 100, rnd, i)
	}

}

func testRandomMILP(t *testing.T, nTest int, pZero float64, maxN int, rnd *rand.Rand, workers int) {
	var sol milpSolution
	var err error

	// Try a bunch of random LPs
	for i := 0; i < nTest; i++ {
		n := rnd.Intn(maxN) + 2 // n must be at least two.
		m := rnd.Intn(n-1) + 1  // m must be between 1 and n
		prob := getRandomMILP(pZero, m, n, rnd)

		fmt.Println("------ problem ", i)
		fmt.Println("c:")
		fmt.Println(prob.c)
		fmt.Println("integrality:")
		fmt.Println(prob.integralityConstraints)
		fmt.Println("A:")
		fmt.Println(mat.Formatted(prob.A))
		fmt.Println("b:")
		fmt.Println(prob.b)
		fmt.Println("G:")
		fmt.Println(mat.Formatted(prob.G))
		fmt.Println("h:")
		fmt.Println(prob.h)

		// assign the solution to prevent the compiler from optimizing the call out
		// note that we are adding a one-second timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		sol, err = prob.solve(ctx, workers, dummyMiddleware{})

		if err != nil {
			t.Log(err)
			t.Log(sol.solution)
		}

	}

}

// adapted from Gonum's lp.Simplex.
func getRandomMILP(pZero float64, m, n int, rnd *rand.Rand) *milpProblem {

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
	return &milpProblem{
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
