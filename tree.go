package ilp

import (
	"fmt"
	"math"
	"sync"

	"gonum.org/v1/gonum/optimize/convex/lp"
)

// The logTree datastructure is used as a log of the branch-and-bound algorithms decisions.
// This code should not contain algorithm business logic to ensure loose coupling.
// Note that we don't want to store references to subproblem datastructures, as this would preclude GC for these potentially large structs.

type logTree struct {
	root *node
}

type node struct {

	// a summary of the solution of this node
	x []float64
	z float64

	// the decision that took place at this node
	decision bnbDecision

	// the node's children, if any.
	children []*node
}

// Branch-and-bound decisions that can be made by the algorithm
type bnbDecision string

const (
	SUBPROBLEM_IS_DEGENERATE        bnbDecision = "subproblem contains a degenerate (singular) matrix"
	SUBPROBLEM_NOT_FEASIBLE         bnbDecision = "subproblem has no feasible solution"
	WORSE_THAN_INCUMBENT            bnbDecision = "worse than incumbent"
	BETTER_THAN_INCUMBENT_BRANCHING bnbDecision = "better than incumbent but infeasible, so branching"
	BETTER_THAN_INCUMBENT_FEASIBLE  bnbDecision = "better than incumbent and feasible, so replacing incumbent"
	INITIAL_RX_FEASIBLE_FOR_IP      bnbDecision = "initial relaxation is feasible for IP"
	INITIAL_RELAXATION_LEGAL        bnbDecision = "initial relaxation is legal"
)

func newLogTree(rootNode *node) *logTree {
	return &logTree{
		root: rootNode,
	}
}

// Note that we do not save a reference to the entire solution struct as this would be a potential GC nightmare.
func newNode(s solution) (n *node) {
	n = &node{
		x:        s.x,
		z:        s.z,
		children: []*node{},
	}
	return
}

func (n *node) addChildren(children ...node) {
	for _, c := range children {
		n.children = append(n.children, &c)
	}
}

func (n *node) setDecision(d bnbDecision) {
	n.decision = d
}

type enumerationTree struct {
	active     chan subProblem
	incumbent  *solution
	candidates chan solution

	// track the number of jobs (solving + checking) currently in progress
	inProgress sync.WaitGroup

	// the root problem
	rootProblem subProblem
}

func newEnumerationTree(rootProblem subProblem) *enumerationTree {
	return &enumerationTree{
		// use a conservatively buffered channel to queue the unsolved problems in
		active:     make(chan subProblem, 10),
		candidates: make(chan solution, 10),

		rootProblem: rootProblem,
	}
}

func (p *enumerationTree) startSearch(nworkers int) (solution, *logTree) {

	// solve the initial relaxation
	initialRelaxationSolution := p.rootProblem.solve()
	if initialRelaxationSolution.err != nil {

		// override the error message in case of infeasible initial relaxation for easier debugging
		if initialRelaxationSolution.err == lp.ErrInfeasible {
			initialRelaxationSolution.err = INITIAL_RELAXATION_NOT_FEASIBLE
		}
		return initialRelaxationSolution, nil
	}

	// initiate the logging tree with the solution to the initial relaxation as the root node
	rootNode := newNode(initialRelaxationSolution)
	tree := newLogTree(rootNode)

	// If no integrality constraints are present, we can return the initial solution as-is if it is feasible.
	// moreover, if the solution to the initial relaxation already satisfies all integrality constraints, we can present it as-is.
	if feasibleForIP(p.rootProblem.integralityConstraints, initialRelaxationSolution.x) {
		return initialRelaxationSolution, tree
	}

	// set the initial relaxation solution as the incumbent
	p.postCandidate(initialRelaxationSolution)

	// start the solve workers
	for j := 0; j < nworkers; j++ {
		go p.solveWorker()
	}

	// start the checker worker
	go p.solutionChecker()

	// wait until there are no longer any jobs active
	p.inProgress.Wait()

	// close the channels feeding the worker goroutines
	close(p.active)
	close(p.candidates)

	return *p.incumbent, tree

}

func (p *enumerationTree) postCandidate(s solution) {
	// inform the manager that we added a candidate to the queue
	p.inProgress.Add(1)
	p.candidates <- s
}

func (p *enumerationTree) enqueueProblems(probs ...subProblem) {
	for _, s := range probs {

		p.inProgress.Add(1)
		p.active <- s
	}
}

func (p *enumerationTree) solveWorker() {
	for prob := range p.active {
		// solve the subproblem
		candidate := prob.solve()

		// present the candidate solution
		p.postCandidate(candidate)

		// tell the manager we finished a unit of work
		p.inProgress.Done()
	}

}

func (p *enumerationTree) solutionChecker() {

	for candidate := range p.candidates {

		if p.incumbent == nil {
			p.incumbent = &candidate

			// branch the inital relaxation and add its children to the queue
			p1, p2 := p.incumbent.branch()

			// add the new problems back into the queue
			p.enqueueProblems(p1, p2)

		} else {
			// decide on what to do with the candidate solution:
			// var decision bnbDecision

			switch {

			case candidate.err != nil:
				// TODO: store this decision
				translateSolverFailure(candidate.err)
				// failure := translateSolverFailure(candidate.err)
				// decision = failure

			// Note that the objective is always minimization.
			case p.incumbent.z <= candidate.z:
				// noop
				// decision = WORSE_THAN_INCUMBENT

			case p.incumbent.z > candidate.z:
				if feasibleForIP(p.rootProblem.integralityConstraints, candidate.x) {
					// candidate is an improvement over the incumbent
					p.incumbent = &candidate
					// decision = BETTER_THAN_INCUMBENT_FEASIBLE
				} else {
					//candidate is an improvement over the incumbent, but not feasible.
					//branch and add the descendants of this candidate to the queue
					// decision = BETTER_THAN_INCUMBENT_BRANCHING
					p1, p2 := candidate.branch()
					p.enqueueProblems(p1, p2)

				}

			default:
				// this should never happen and thus should never fail silently.
				// Leave this here in case anything is every screwed up in the case logic that would make this case reachable.
				panic("unexpected case: could not decide what to do with branched subproblem")

			}
		}

		// inform the manager that we finished checking a candidate
		p.inProgress.Done()
	}

}

// takes a solver failure and determines whether it warrants a panic or whether it is expected.
func translateSolverFailure(err error) bnbDecision {
	for failure, decision := range expectedFailures {
		if failure == err {
			return decision
		}
	}
	panic(err)
}

// check whether the solution vector is feasible in light of the integrality constraints for each variable
func feasibleForIP(constraints []bool, solution []float64) bool {
	if len(constraints) != len(solution) {
		panic(fmt.Sprint("constraints vector and solution vector not of equal size: ", constraints, solution))
	}
	for i := range solution {
		if constraints[i] {
			if !isAllInteger(solution[i]) {
				return false
			}
		}
	}
	return true
}

func isAllInteger(in ...float64) bool {
	for _, k := range in {
		if !(k == math.Trunc(k)) {
			return false
		}
	}
	return true
}
