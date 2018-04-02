package ilp

import (
	"reflect"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func Test_RemoveEmptyRows(t *testing.T) {
	type args struct {
		A *mat.Dense
		b []float64
	}
	tests := []struct {
		name string
		args args
		Anew *mat.Dense
		bNew []float64
	}{
		{
			name: "no empty rows",
			args: args{
				A: mat.NewDense(4, 4, []float64{
					0, 1, 1, 1,
					2, 0, 0, 0,
					3, 0, 0, 0,
					0, 0, 1, 0,
				}),
				b: []float64{1, 2, 3, 4},
			},
			Anew: mat.NewDense(4, 4, []float64{
				0, 1, 1, 1,
				2, 0, 0, 0,
				3, 0, 0, 0,
				0, 0, 1, 0,
			}),
			bNew: []float64{1, 2, 3, 4},
		},
		{
			name: "one empty row",
			args: args{
				A: mat.NewDense(4, 4, []float64{
					0, 1, 1, 1,
					0, 0, 0, 0,
					3, 0, 0, 0,
					0, 0, 1, 0,
				}),
				b: []float64{1, 2, 3, 4},
			},
			Anew: mat.NewDense(3, 4, []float64{
				0, 1, 1, 1,
				3, 0, 0, 0,
				0, 0, 1, 0,
			}),
			bNew: []float64{1, 3, 4},
		},
		{
			name: "two empty rows",
			args: args{
				A: mat.NewDense(4, 4, []float64{
					0, 1, 1, 1,
					0, 0, 0, 0,
					3, 0, 0, 0,
					0, 0, 0, 0,
				}),
				b: []float64{1, 2, 3, 4},
			},
			Anew: mat.NewDense(2, 4, []float64{
				0, 1, 1, 1,
				3, 0, 0, 0,
			}),
			bNew: []float64{1, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA, gotB := removeEmptyRows(tt.args.A, tt.args.b)
			if !reflect.DeepEqual(gotA, tt.Anew) {
				t.Errorf("removeEmptyRows() got = %v, want %v", gotA, tt.Anew)
			}
			if !reflect.DeepEqual(gotB, tt.bNew) {
				t.Errorf("removeEmptyRows() got1 = %v, want %v", gotB, tt.bNew)
			}
		})
	}
}
