package ilp

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"

	"gonum.org/v1/gonum/optimize/convex/lp"
)

// The logTree datastructure is used as a log of the branch-and-bound algorithms decisions.
// This code should not contain algorithm business logic to ensure loose coupling.
// Note that we don't want to store references to subproblem datastructures, as this would preclude GC for these potentially large structs.
// TODO: add methods and actual logging functionality
// TODO: maybe use an interface for easier instrumentation during testing
// Note that we should take care not to (indirectly) save a reference to the entire subProblem struct as this would be a potential GC nightmare.
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
	BETTER_THAN_INCUMBENT_BRANCHING bnbDecision = "better than incumbent but not integer feasible, so branching"
	BETTER_THAN_INCUMBENT_FEASIBLE  bnbDecision = "better than incumbent and integer feasible, so replacing incumbent"
	INITIAL_RX_FEASIBLE_FOR_IP      bnbDecision = "initial relaxation is feasible for IP"
	INITIAL_RELAXATION_LEGAL        bnbDecision = "initial relaxation is legal and thus set as initial incumbent"
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

type enumerationTree struct {
	active     chan subProblem
	toSolve    chan subProblem
	incumbent  *solution
	candidates chan solution

	// track the number of jobs (solving + checking) currently in progress
	workInProgress int64

	// the root problem
	rootProblem subProblem
}

func newEnumerationTree(rootProblem subProblem) *enumerationTree {
	return &enumerationTree{
		// do not build buffered channels: buffering is managed by a separate goroutine.
		active:     make(chan subProblem),
		toSolve:    make(chan subProblem),
		candidates: make(chan solution),

		rootProblem: rootProblem,
	}
}

func (p *enumerationTree) startSearch(nworkers int, ctx context.Context) (*solution, *logTree) {

	// solve the initial relaxation
	initialRelaxationSolution := p.rootProblem.solve()
	if initialRelaxationSolution.err != nil {

		// override the error message in case of infeasible initial relaxation for easier debugging
		if initialRelaxationSolution.err == lp.ErrInfeasible {
			initialRelaxationSolution.err = INITIAL_RELAXATION_NOT_FEASIBLE
		}
		return &initialRelaxationSolution, nil
	}

	// initiate the logging tree with the solution to the initial relaxation as the root node
	rootNode := newNode(initialRelaxationSolution)
	tree := newLogTree(rootNode)

	// If no integrality constraints are present, we can return the initial solution as-is if it is feasible.
	// moreover, if the solution to the initial relaxation already satisfies all integrality constraints, we can present it as-is.
	if feasibleForIP(p.rootProblem.integralityConstraints, initialRelaxationSolution.x) {
		return &initialRelaxationSolution, tree
	}

	// start the buffer pump that manages transfers of subProblems from the buffer to the worker pool
	go p.bufferManager()

	// start the solve workers
	for j := 0; j < nworkers; j++ {
		go p.solveWorker()
	}

	// check the initial relaxation solution
	p.checkSolution(initialRelaxationSolution)

	// listen for new candidates to check but also keep an eye out for any cancellation signals.
mainWait:
	for atomic.LoadInt64(&p.workInProgress) > 0 {
		select {
		case candidate := <-p.candidates:
			p.checkSolution(candidate)
			p.workDone()
		case <-ctx.Done():
			//TODO: log that the context timeout expired
			break mainWait
		}
	}

	// close the channels feeding the buffer pump, which will cause the downstream goroutines to return.
	close(p.toSolve)

	// The incumbent can still be nil. This can happen for instance when the context stops the search early.
	return p.incumbent, tree

}

func (p *enumerationTree) postCandidate(s solution) {
	p.candidates <- s
}

func (p *enumerationTree) enqueueProblems(probs ...subProblem) {
	for _, s := range probs {

		p.workAdded()

		p.toSolve <- s

	}
}

func (p *enumerationTree) workAdded() {
	atomic.AddInt64(&p.workInProgress, 1)
}

func (p *enumerationTree) workDone() {
	atomic.AddInt64(&p.workInProgress, -1)
}

// bufferManager should run in a separate goroutine to prevent blocking of the communication between the solvers and the checker
func (p *enumerationTree) bufferManager() {
	var buffer []subProblem
	var next subProblem

	// key feature of the statement below is the exploitation of nil channels. Select skips over these.
	var output chan subProblem

loopy:
	for {

		select {

		// if presented, store the piece of work in the buffer.
		case msg, open := <-p.toSolve:
			if !open {
				// if the buffer channel is closed, we exit the loop
				break loopy
			}
			buffer = append(buffer, msg)

		// try to send a buffered job to the workers
		// note that when next is nil, so is the output channel. A nil channel causes select to skip over this case.
		case output <- next:
			// pop the buffered job that we just sent (only if it WAS sent, ofcourse)
			if len(buffer) > 1 {
				buffer = buffer[1:]
			} else {
				buffer = nil
			}

		}

		if len(buffer) > 0 {
			next = buffer[0]
			output = p.active
		} else {
			output = nil
		}

	}

	// After the loop has been broken, we close the buffer channels.
	close(p.active)
}

func (p *enumerationTree) solveWorker() {
	for prob := range p.active {
		// solve the subproblem
		candidate := prob.solve()

		// present the candidate solution
		p.postCandidate(candidate)
	}

}

// TODO: store each decision somewhere
func (p *enumerationTree) checkSolution(candidate solution) {

	// decide on what to do with the candidate solution:
	// var decision bnbDecision

	// retrieve the objective function value of the incumbent
	// if no incumbent is set, return +Inf
	incumbentZ := math.Inf(1)
	if p.incumbent != nil {
		incumbentZ = p.incumbent.z
	}

	switch {

	case candidate.err != nil:
		translateSolverFailure(candidate.err)
		// failure := translateSolverFailure(candidate.err)
		// decision = failure

	// Note that the objective is always minimization.
	case incumbentZ <= candidate.z:
		// noop
		// decision = WORSE_THAN_INCUMBENT

	case incumbentZ > candidate.z:
		if feasibleForIP(p.rootProblem.integralityConstraints, candidate.x) {
			// Candidate is an improvement over the incumbent
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
