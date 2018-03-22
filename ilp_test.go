package ilp

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/mat"
)

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

	solution := s.solve()
	t.Log(solution.problem)
	fmt.Println(solution.x)
	assert.NoError(t, solution.err)
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
			want: MILPsolution{
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
			want: MILPsolution{
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
			want: MILPsolution{
				solution: solution{
					x: []float64{1.0674157303370786, 2.3595505617977524, 0},
					z: -5.786516853932583,
				},
			},
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
			if err != tt.wantErr {
				t.Log(got)
				t.Errorf("MILPproblem.Solve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Note: we compare only the numerical solution variables
			if !(reflect.DeepEqual(tt.want.solution.x, got.solution.x) && tt.want.solution.z == got.solution.z) {
				t.Log(got)
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

		// fmt.Println("c:")
		// fmt.Println(prob.c)
		// fmt.Println("A:")
		// fmt.Println(mat.Formatted(prob.A))
		// fmt.Println("b:")
		// fmt.Println(prob.b)
		prob.Solve()

		// fmt.Println(solution.solution.x, solution.solution.z, err)
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
