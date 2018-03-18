package ilp

import "math"

// selectable heuristic options
type BranchHeuristic int

const (
	BRANCH_MAXFUN          BranchHeuristic = 0
	BRANCH_MOST_INFEASIBLE BranchHeuristic = 1
	BRANCH_NAIVE           BranchHeuristic = 2
)

// Get the variable to branch on by looking at which variables we branched on previously.
// If there are no branches yet, so we start at the first constrained variable.
// Note that this is a really naive way to find a nice variable to branch on.
func (s solution) naiveBranchPoint() int {
	branchOn := 0

	// if there are branches, we cycle through the variables starting from the last one we branched on
	// when we encounter the next variable with an integrality constraint, we pick that one to branch on.
	if len(s.problem.bnbConstraints) == 0 {
		for i := range s.problem.integralityConstraints {
			if s.problem.integralityConstraints[i] {
				branchOn = i
			}
		}
	} else {

		// Get the last variable we branched.
		lastConstraint := s.problem.bnbConstraints[len(s.problem.bnbConstraints)-1]
		lastBranchedVariable := lastConstraint.branchedVariable

		// increment this variable until we encounter the next constrained variable or we reach the end of the variable vector.
		cursor := lastBranchedVariable
		for {
			if cursor == len(s.problem.c)-1 {
				// we bring the cursor back to the beginning
				cursor = -1
			}
			cursor++
			if s.problem.integralityConstraints[cursor] {
				branchOn = cursor
				break
			}
		}

	}

	return branchOn
}

// // Choose the integrality-constrained variable with the highest absolute value in the objective function
func maxFunBranchPoint(c []float64, integralityConstraints []bool) int {
	if len(c) != len(integralityConstraints) {
		panic("number of variables not equal to number of integrality constraints")
	}

	var candidateValue float64
	currentCandidate := 0

	for i, v := range c {
		if integralityConstraints[i] {
			// we use greater-than-or-equal-to to ensure an integer-constrained variable is selected if one is present, even if its coefficient is 0.
			if math.Abs(v) >= candidateValue {
				currentCandidate = i
			}
		}
	}

	return currentCandidate
}

// Choose the variable with the fractional part closest to 1/2.
func mostInfeasibleBranchPoint(c []float64, integralityConstraints []bool) int {
	if len(c) != len(integralityConstraints) {
		panic("number of variables not equal to number of integrality constraints")
	}

	candidateRemainder := 1.0
	currentCandidate := 0

	for i, v := range c {
		if integralityConstraints[i] {
			_, f := math.Modf(v)
			// we use greater-than-or-equal-to to ensure an integer-constrained variable is selected if one is present, even if candidate value is equal to 0.
			if (0.5 - f) <= candidateRemainder {
				currentCandidate = i
			}
		}
	}

	return currentCandidate
}
