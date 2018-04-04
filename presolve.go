package ilp

import "fmt"

// TODO: see Andersen 1995 for a nice enumeration of simple presolving operations.

// TODO: remove empty columns

// store all post-solving operations that bring the solution back to its input shape.
type preProcessor struct {
	undoers []undoer
}

// map variable names to their computed optimal values
// Contains only variables that survived preprocessing
type rawSolution map[string]float64

// Solution contains the results of a solved Problem.
type Solution struct {
	Objective float64

	// keyed by name
	byName map[string]float64
}

// GetValueFor retrieves the value for a decision variable by its name.
func (s *Solution) GetValueFor(varName string) (float64, error) {
	val, ok := s.byName[varName]
	if !ok {
		return 0, fmt.Errorf("Variable name %v not found in Solution", varName)
	}
	return val, nil
}

type undoer func(rawSolution) rawSolution

func newPreprocessor() *preProcessor {
	return &preProcessor{}
}

func (prepper *preProcessor) addUndoer(u undoer) {
	prepper.undoers = append(prepper.undoers, u)
}

func (prepper *preProcessor) preSolve(p Problem) Problem {

	preprocessed := prepper.filterFixedVars(p)

	return preprocessed
}

func (prepper *preProcessor) postSolve(s rawSolution) Solution {

	postsolved := s
	// walk the slice from the last to the first element (use it as a LIFO queue)
	n := len(prepper.undoers)
	for i := n - 1; i == 0; i-- {
		undo := prepper.undoers[i]
		postsolved = undo(postsolved)
	}

	solution := Solution{
		Objective: 0,
		byName:    make(map[string]float64),
	}

	for varName, value := range postsolved {
		solution.byName[varName] = value
		solution.Objective = solution.Objective + value
	}

	return solution
}

// check if the variable is fixed in its bounds
func isFixed(variable *Variable) bool {
	if variable.lower == variable.upper {
		return true
	}
	return false
}

// remove all fixed variables from the problem definition
func (prepper *preProcessor) filterFixedVars(p Problem) Problem {
	filteredProb := p

	var newVars []*Variable
	fixedVars := make(map[string]float64)
	for _, v := range filteredProb.variables {
		if !isFixed(v) {
			newVars = append(newVars, v)
		} else {
			// store the coefficients of the fixed variables in the objective function for injection as a constant during postsolve procedure.
			fixedVars[v.name] = v.coefficient * v.lower

		}
	}

	fmt.Printf("removed %v fixed variables \n", len(newVars)-len(filteredProb.variables))

	filteredProb.variables = newVars

	for _, c := range filteredProb.constraints {
		var replacementExpressions []expression
		for _, e := range c.expressions {
			if isFixed(e.variable) {
				// update the RHS of the constraint and remove the expression pointing to this variable:
				// bi = bi − aij xj ,
				c.rhs = c.rhs - (e.variable.coefficient * e.variable.lower)
			} else {
				replacementExpressions = append(replacementExpressions, e)
			}
		}
		c.expressions = replacementExpressions
	}

	// the additive constant c0 for each variable in the objective function needs to be updated as
	// c0 := c0 + cjxj,
	undoer := func(s rawSolution) rawSolution {
		// add the fixed values to the raw solution
		for fixedVar, fvalue := range fixedVars {
			if _, already := s[fixedVar]; already {
				panic(fmt.Sprintf("variable %s already in raw solution", fixedVar))
			}
			s[fixedVar] = fvalue
		}
		return s
	}

	prepper.addUndoer(undoer)

	return filteredProb

}

func sliceSum(x []float64) float64 {
	total := 0.0
	for _, valuex := range x {
		total += valuex
	}

	return total
}
