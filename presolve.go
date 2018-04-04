package ilp

import (
	"fmt"

	"github.com/deckarep/golang-set"
)

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

	fmt.Printf("Presolving problem with %v variables and %v constraints\n", len(p.variables), len(p.constraints))

	// remove redundancies caused by the user.
	preprocessed := sanitizeProblem(p)

	// loop over the prepping operations until no more modifications are performed
	previousNUndoers := 0
presolve:
	for {
		preprocessed = prepper.filterFixedVars(preprocessed)
		preprocessed = prepper.findImplicitlyFixedVars(preprocessed)
		preprocessed = removeEmptyConstraints(preprocessed)
		preprocessed = removeDuplicateConstraints(preprocessed)

		if len(prepper.undoers) == previousNUndoers {
			break presolve
		}
		previousNUndoers = len(prepper.undoers)
	}

	fmt.Println("presolve done")

	fmt.Printf("Presolving reduced problem to %v variables and %v constraints\n", len(preprocessed.variables), len(preprocessed.constraints))

	return preprocessed
}

func (prepper *preProcessor) postSolve(s rawSolution) Solution {

	postsolved := s
	// walk the slice from the last to the first element (use it as a LIFO queue)
	n := len(prepper.undoers)
	for i := n - 1; i >= 0; i-- {
		postsolved = prepper.undoers[i](postsolved)
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

// remove redundant statements from the problem definition that were introduced by the user.
// TODO: explicit duplicate constraints
// TODO: constraints that are superseded by the variable bounds?
func sanitizeProblem(p Problem) Problem {
	for _, c := range p.constraints {
		c.expressions = filterZeroExpressions(c.expressions)
	}

	return p
}

func filterZeroExpressions(exprs []expression) []expression {
	var nonzero []expression
	for _, e := range exprs {
		if e.coef != 0 {
			nonzero = append(nonzero, e)
		}
	}
	return nonzero
}

// check if the variable is fixed in its bounds
func isFixed(variable *Variable) bool {
	if variable.lower == variable.upper {
		return true
	}
	return false
}

// remove all fixed variables from the problem definition
// TODO: try to also find variables that are fixed in the constraint definitions (currently only looking at explicitly defined variable bounds)
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

	fmt.Printf("removed %v fixed variables \n", len(filteredProb.variables)-len(newVars))
	filteredProb.variables = newVars

	// update the RHS of the constraint and remove the expression pointing to this variable:
	// bi = bi − aij xj ,
	for _, c := range filteredProb.constraints {
		var replacementExpressions []expression
		for _, e := range c.expressions {
			if isFixed(e.variable) {
				c.rhs = c.rhs - (e.variable.coefficient * e.variable.lower)
			} else {
				replacementExpressions = append(replacementExpressions, e)
			}
		}
		c.expressions = replacementExpressions
	}

	// the additive constant c0 for each variable in the objective function needs to be updated as
	// c0 := c0 + cjxj,
	if len(fixedVars) > 0 {
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
	}

	return filteredProb

}

// all variables that are implicitly fixed due to the shape of a constraint should be set to be explicitly fixed.
// Note that this could be part of a second pass; setting the implicitly fixed vars to explicitly fixed and then removing them with filterFixedVars.
// TODO: However, we dont want to modify the original variables (i.e. set their bounds)
// TODO: a more elegant procedure can be considered. This procedure only considers constraint i with bi = 0 and Sij > 0, making it very limited in its application.
func (prepper *preProcessor) findImplicitlyFixedVars(p Problem) Problem {

	implicitZero := make(map[*Variable]struct{})
	for _, c := range p.constraints {
		removable := false
		if c.rhs == 0 {

			// check for any negative coefficients
			nonnegative := true
		checker:
			for _, e := range c.expressions {
				if e.coef < 0 {
					nonnegative = false
					break checker
				}
			}

			if nonnegative {
				removable = true
			}
		}

		if removable {
			for _, e := range c.expressions {
				// be careful not to consider variables in expressions with coef 0 (dummies) as implicit zero.
				// This is basically a double-check: these expressions should already be removed during Problem sanitation
				if e.coef > 0 {
					implicitZero[e.variable] = struct{}{}
				}
			}
		}
	}

	fmt.Printf("found %v variables implicitly fixed at zero \n", len(implicitZero))
	//TODO: MODIFIES ORIGINAL PROBLEM: REMOVE ME (just a PoC)
	for v := range implicitZero {
		v.LowerBound(0).UpperBound(0)
	}

	return p
}

// constraints can turn empty after earlier variable-centric preprocessing operations. These should be removed.
func removeEmptyConstraints(p Problem) Problem {
	var filtered []*Constraint
	for _, c := range p.constraints {
		if len(c.expressions) > 0 {
			filtered = append(filtered, c)
		}
	}

	fmt.Printf("removed %v empty constraints\n", len(p.constraints)-len(filtered))
	p.constraints = filtered
	return p
}

// This function may need a rethink if this turns out not to be performant for larger problems.
func removeDuplicateConstraints(p Problem) Problem {

	// map each set that uniquely identifies each constraint to the Constraint
	var sets []mapset.Set
	for _, constraint := range p.constraints {

		// add the variable names of the constraint to a set
		cSet := mapset.NewSet()

		for _, e := range constraint.expressions {
			cSet.Add(fmt.Sprintf("%v-%v", e.variable.name, e.coef))
		}

		sets = append(sets, cSet)
	}

	// cross-compare each set of variable-coefficient expressions.
	// if we encounter a duplicate, we throw the one with the highest rhs out.
	var equalExpressions [][]*Constraint
	var retained []*Constraint
	for i, s := range sets {
		unique := true
		for j := range sets {
			if i == j {
				continue
			}
			if sets[j].Equal(s) {
				equalExpressions = append(equalExpressions, []*Constraint{p.constraints[i], p.constraints[j]})
				unique = false
			}
		}
		if unique {
			retained = append(retained, p.constraints[i])
		}
	}

	// decide which one of the non-unique pairs to retain
	// -> pick the one with the smallest RHS, regardless of it being an equality or inequality.
	for _, pairs := range equalExpressions {
		if pairs[0].rhs > pairs[1].rhs {
			retained = append(retained, pairs[1])
		}
	}

	fmt.Printf("removed %v (%v) duplicated constraints \n", len(equalExpressions), len(p.constraints)-len(retained))

	// substitute the constraints slice
	p.constraints = retained

	return p

}
