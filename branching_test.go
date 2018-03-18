package ilp

import (
	"testing"
)

func Test_maxFunBranchPoint(t *testing.T) {
	type args struct {
		c                      []float64
		integralityConstraints []bool
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "no integrality constraints",
			args: args{
				c: []float64{1, 2, 3, 4, 5},
				integralityConstraints: []bool{false, false, false, false, false},
			},
			want: 0,
		},
		{
			name: "one integrality constraint",
			args: args{
				c: []float64{1, 2, 3, 4, 5},
				integralityConstraints: []bool{false, false, true, false, false},
			},
			want: 2,
		},
		{
			name: "one integrality constraint, but no improvement over 0",
			args: args{
				c: []float64{1, 2, 0, 4, 5},
				integralityConstraints: []bool{false, false, true, false, false},
			},
			want: 2,
		},
		{
			name: "multiple integrality constraints, differing values",
			args: args{
				c: []float64{1, 2, 3, 4, 5},
				integralityConstraints: []bool{true, true, true, true, false},
			},
			want: 3,
		},
		{
			name: "multiple integrality constraints, similar values",
			args: args{
				c: []float64{1, 2, 4, 4, 5},
				integralityConstraints: []bool{true, true, true, true, false},
			},
			want: 3,
		},
		{
			name: "all integrality constraints, similar values",
			args: args{
				c: []float64{1, 2, 4, 4, 5},
				integralityConstraints: []bool{true, true, true, true, true},
			},
			want: 4,
		},
		{
			name: "negative coefficients",
			args: args{
				c: []float64{1, 2, 4, 4, -5},
				integralityConstraints: []bool{true, true, true, true, true},
			},
			want: 4,
		},
		{
			name: "multiple equal negative coefficients",
			args: args{
				c: []float64{1, 2, 4, -5, -5},
				integralityConstraints: []bool{true, true, true, true, true},
			},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maxFunBranchPoint(tt.args.c, tt.args.integralityConstraints); got != tt.want {
				t.Errorf("maxFunBranchPoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_closestFractionalBranchPoint(t *testing.T) {
	type args struct {
		c                      []float64
		integralityConstraints []bool
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "No obvious candidate",
			args: args{
				c: []float64{1, 2, 3, 4},
				integralityConstraints: []bool{false, true, true, true},
			},
			want: 3,
		},
		{
			name: "No constraints",
			args: args{
				c: []float64{1, 2, 3, 4},
				integralityConstraints: []bool{false, false, false, false},
			},
			want: 0,
		},
		{
			name: "obvious candidate",
			args: args{
				c: []float64{1, 2, 3, 4.6},
				integralityConstraints: []bool{false, true, true, true},
			},
			want: 3,
		},
		{
			name: "obvious candidate < 0.5",
			args: args{
				c: []float64{1, 2, 3, 4.2},
				integralityConstraints: []bool{false, true, true, true},
			},
			want: 3,
		},
		{
			name: "exact match on 1/2",
			args: args{
				c: []float64{1, 2, 3, 4.5},
				integralityConstraints: []bool{false, true, true, true},
			},
			want: 3,
		},
		{
			name: "multiple exact matches on 1/2",
			args: args{
				c: []float64{1, 2, 3.5, 4.5},
				integralityConstraints: []bool{false, true, true, true},
			},
			want: 3,
		},
		{
			name: "multiple exact matches on 1/2. Should pick latest.",
			args: args{
				c: []float64{1, 2, 3.5, 4.5},
				integralityConstraints: []bool{false, true, true, true},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mostInfeasibleBranchPoint(tt.args.c, tt.args.integralityConstraints); got != tt.want {
				t.Errorf("closestFractionalBranchPoint() = %v, want %v", got, tt.want)
			}
		})
	}
}
