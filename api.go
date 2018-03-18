package ilp

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// TODO: sanity checks before converting Problem to a MILPproblem, such as NaN, Inf, and matrix shapes and variable bound domains
// TODO: parsing of variable bounds to constraints does not deal with negative domains
// TODO: GLPK testing is extremely convoluted due to its shitty API. Moreover, its output is sometimes plain wrong (doesnt diagnose unbounded problems).
// TODO: try to formulate more advanced constraints, like sets of values instead of just integrality?
// Note that having integer sets as constraints is basically the same as having an integrality constraint, and a <= and >= bound.
// Branching on this type of constraint can be optimized in a neat way (i.e. x>=0, x<=1, x<=0 ~-> x = 0)
// TODO: dealing with variables that are unrestricted in sign (currently, each var is subject to a nonnegativity constraint)
// TODO: make CLI and Problem serialization format for easy integration with R/python-based analysis tooling for debugging of mathematical properties.
// TODO: explore options regarding branch-and-bound parallelism. See also computation of (pseudo-)costs and expensive branching heuristics.
// TODO: make solver cancellable with a context by spinning the actual solving off into a goroutine.
// TODO: add check for when adding a constraint: check whether an expression containing that variable already exists.
// TODO: write benchmarks for time and space usage
// TODO: small(?) performance gains may be made by switching dense matrix datastructures over to sparse for big problems

// The abstract MILP problem representation
type Problem struct {
	// minimizes by default
	maximize bool

	// the problem structure
	variables    []*Variable
	inequalities []Inequality
	equalities   []Equality

	// the branching heuristic to use for branch-and-bound (defaults to 0 == maxFun)
	branchingHeuristic BranchHeuristic
}

// A variable of the MILP problem.
type Variable struct {
	// variable name for human reference
	name string

	// coefficient of the variable in the objective function
	coefficient float64

	// integrality constraint
	integer bool

	// bounds
	upper float64
	lower float64
}

// an Expression of a variable and an arbitrary float for use in defining constraints
// e.g. "-1 * x1"
type Expression struct {
	coef     float64
	variable *Variable
}

// An abstraction representing an inequality constraint.
type Inequality struct {
	// expressions will be summed together to form the LHS of ...
	expressions []Expression

	// ... a constraint with a certain RHS
	smallerThan float64
}

// An abstraction representing an equality constraint.
type Equality struct {
	// expressions will be summed together to form the LHS of ...
	expressions []Expression

	// ... a constraint with a certain RHS
	equalTo float64
}

// Initiate a new MILP problem abstraction
func NewProblem() Problem {
	return Problem{}
}

// add a variable and return a reference to that variable.
// Defaults to no integrality constraint and an objective function coefficient of 0
func (p *Problem) AddVariable(name string) *Variable {

	v := Variable{
		name:        name,
		coefficient: 0,
		integer:     false,
		upper:       math.Inf(1),
		lower:       0,
	}

	p.variables = append(p.variables, &v)

	return &v
}

// SetCoeff sets the value of the variable in the objective function
func (v *Variable) SetCoeff(coef float64) *Variable {
	v.coefficient = coef
	return v
}

func (v *Variable) IsInteger() *Variable {
	v.integer = true
	return v
}

// UpperBound sets the inclusive upper bound of this variable. Input must be positive.
func (v *Variable) UpperBound(bound float64) *Variable {
	v.upper = bound
	return v
}

// LowerBound sets the inclusive lower bound of this variable. Input must be positive.
func (v *Variable) LowerBound(bound float64) *Variable {
	v.lower = bound
	return v
}

// Add an Equality constraint to the problem, given a set of expressions that must equal equalTo.
func (p *Problem) AddEquality(expr []Expression, equalTo float64) {
	if len(expr) == 0 {
		panic("must add expressions")
	}

	for _, e := range expr {
		if !p.checkExpression(e) {
			panic("provided expression contains a variable that has not been declared to this problem yet")
		}
	}

	p.equalities = append(p.equalities, Equality{
		expressions: expr,
		equalTo:     equalTo,
	})

}

// Add an InEquality constraint to the problem, given a set of expressions that must be less than smallerThan.
func (p *Problem) AddInEquality(expr []Expression, smallerThan float64) {
	if len(expr) == 0 {
		panic("must add expressions")
	}

	for _, e := range expr {
		if !p.checkExpression(e) {
			panic("provided expression contains a variable that has not been declared to this problem yet")
		}
	}

	p.inequalities = append(p.inequalities, Inequality{
		expressions: expr,
		smallerThan: smallerThan,
	})

}

func (p *Problem) Maximize() {
	p.maximize = true
}

func (p *Problem) Minimize() {
	p.maximize = false
}

func (p *Problem) BranchingHeuristic(choice BranchHeuristic) {
	p.branchingHeuristic = choice
}

// Check whether the expression is legal considering the variables currently present in the problem
func (p *Problem) checkExpression(e Expression) bool {

	// check whether the pointer to the variable provided is currently included in the Problem
	for _, v := range p.variables {
		if v == e.variable {
			return true
		}
	}

	return false

}

// get the index of the variable pointer in the variable pointer slice of the Problem struct using a linear search
func (p *Problem) getVariableIndex(v *Variable) int {
	for i, va := range p.variables {
		if v == va {
			return i
		}
	}
	panic("variable pointer not found in Problem struct")
}

// Convert the abstract problem representation to its concrete numerical representation.
func (p *Problem) ToSolveable() *MILPproblem {

	// get the c vector containing the coefficients of the variables in the objective function
	// simultaneously parse the integrality constraints
	var c []float64
	var integrality []bool
	for _, v := range p.variables {

		// if the Problem is set to be maximized, we assume that all variable coefficients reflect that.
		// To turn this maximization problem into a minimization one, we multiply all coefficients with -1.
		k := v.coefficient
		if p.maximize {
			k = k * -1
		}

		c = append(c, k)
		integrality = append(integrality, v.integer)
	}

	// add the equality constraints specified as []Expression
	var b []float64
	var Adata []float64
	for _, equality := range p.equalities {

		// build the matrix row for the equality
		equalityRow := make([]float64, len(p.variables))

		for _, exp := range equality.expressions {
			i := p.getVariableIndex(exp.variable)
			equalityRow[i] = exp.coef
		}

		Adata = append(Adata, equalityRow...)

		// add the RHS of the equality to the b vector
		b = append(b, equality.equalTo)
	}

	// combine the Adata vector into a matrix
	var A *mat.Dense
	if len(p.equalities) > 0 {
		A = mat.NewDense(len(p.equalities), len(p.variables), Adata)
	}

	// add the inequality constraints specified as []Expression
	var h []float64
	var Gdata []float64

	for _, inEquality := range p.inequalities {
		inEqualityRow := make([]float64, len(p.variables))

		for _, exp := range inEquality.expressions {
			i := p.getVariableIndex(exp.variable)
			inEqualityRow[i] = exp.coef
		}

		Gdata = append(Gdata, inEqualityRow...)

		// add the RHS of the inequality to the h vector
		h = append(h, inEquality.smallerThan)

	}

	// add the variable bounds as inequality constraints
	for _, v := range p.variables {

		// convert the upper bound to a row in the constraint matrix
		if !math.IsInf(v.upper, 1) {
			uRow := make([]float64, len(p.variables))
			i := p.getVariableIndex(v)
			uRow[i] = 1

			Gdata = append(Gdata, uRow...)

			// add the RHS of the inequality to the h vector
			h = append(h, v.upper)
		}

		// convert the lower bound to a row in the constraint matrix
		if !(v.lower <= 0) {
			uRow := make([]float64, len(p.variables))
			i := p.getVariableIndex(v)
			uRow[i] = -1

			Gdata = append(Gdata, uRow...)

			// add the RHS of the inequality to the h vector
			h = append(h, -v.lower)
		}

	}

	// combine the Gdata vector into a matrix
	var G *mat.Dense
	if len(h) > 0 {
		G = mat.NewDense(len(h), len(p.variables), Gdata)
	}

	return &MILPproblem{
		c: c,
		A: A,
		b: b,
		G: G,
		h: h,
		integralityConstraints: integrality,
		branchingHeuristic:     p.branchingHeuristic,
	}
}
