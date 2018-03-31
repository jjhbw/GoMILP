package ilp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_TreeLogger(t *testing.T) {

	// initiate a treelogger
	tl := NewTreeLogger()

	// add some nodes to the tree
	s1 := solution{
		problem: &subProblem{
			id:     0,
			parent: 0,
			c:      []float64{0, 1},
			A:      nil,
			b:      []float64{0, 1},
			h:      []float64{0, 1},
			integralityConstraints: []bool{false, true},
			branchHeuristic:        0,
		},
		x:   []float64{1, 2},
		z:   float64(1.1),
		err: nil,
	}
	s2 := solution{
		problem: &subProblem{
			id:     1,
			parent: 0,
			c:      []float64{0, 1},
			A:      nil,
			b:      []float64{0, 1},
			h:      []float64{0, 1},
			integralityConstraints: []bool{false, true},
			branchHeuristic:        0,
		},
		x:   []float64{1, 2},
		z:   float64(1.1),
		err: nil,
	}
	s3 := solution{
		problem: &subProblem{
			id:     2,
			parent: 0,
			c:      []float64{0, 1},
			A:      nil,
			b:      []float64{0, 1},
			h:      []float64{0, 1},
			integralityConstraints: []bool{false, true},
			branchHeuristic:        0,
		},
		x:   []float64{1, 2},
		z:   float64(1.1),
		err: nil,
	}

	// tell the logger about the problems
	tl.NewSubProblem(*s1.problem)
	tl.NewSubProblem(*s2.problem)
	tl.NewSubProblem(*s3.problem)

	// tell the logger about the corresponding decisions
	tl.ProcessDecision(s1, BETTER_THAN_INCUMBENT_BRANCHING)
	tl.ProcessDecision(s2, SUBPROBLEM_NOT_FEASIBLE)
	tl.ProcessDecision(s3, SUBPROBLEM_NOT_FEASIBLE)

	// check if the internal node representation looks the way we expect.
	assert.Equal(t, map[int64]node{
		s1.problem.id: node{
			id:       s1.problem.id,
			parent:   s1.problem.parent,
			z:        s1.z,
			x:        s1.x,
			decision: BETTER_THAN_INCUMBENT_BRANCHING,
			solved:   true,
		},
		s2.problem.id: node{
			id:       s2.problem.id,
			parent:   s2.problem.parent,
			z:        s2.z,
			x:        s2.x,
			decision: SUBPROBLEM_NOT_FEASIBLE,
			solved:   true,
		},
		s3.problem.id: node{
			id:       s3.problem.id,
			parent:   s3.problem.parent,
			z:        s3.z,
			x:        s3.x,
			decision: SUBPROBLEM_NOT_FEASIBLE,
			solved:   true,
		},
	}, tl.nodes)

	// var buffer bytes.Buffer
	// tl.toDOT(&buffer)

	// fmt.Println()

	// err := ioutil.WriteFile("enumtree.test.dot", buffer.Bytes(), 0644)
	// assert.NoError(t, err)
}
