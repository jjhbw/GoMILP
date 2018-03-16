package ilp

import (
	"gonum.org/v1/gonum/mat"
)

// TODO: try to formulate more advanced constraints, like sets of values instead of just integrality.
// Note that having integer sets as constraints is basically the same as having an integrality constraint, and a <= and >= bound.
// Branching on this type of constraint can be optimized in a neat way (i.e. x>=0, x<=1, x<=0 ~-> x = 0)
// TODO: dealing with variables that are unrestricted in sign (currently, each var is subject to a nonnegativity constraint)

// The abstract MILP problem representation
type Problem struct {
	// minimizes by default
	maximize     bool
	variables    []*Variable
	inequalities []Inequality
	equalities   []Equality
}

// A variable of the MILP problem.
type Variable struct {
	// coefficient of the variable in the objective function
	Coefficient float64

	// integrality constraint
	Integer bool
}

// an expression of a variable and an arbitrary float for use in defining constraints
// e.g. "-1 * x1"
type expression struct {
	coef     float64
	variable *Variable
}

// An abstraction representing an inequality constraint.
type Inequality struct {
	// expressions will be summed together to form the LHS of ...
	expressions []expression

	// ... a constraint with a certain RHS
	smallerThan float64
}

// An abstraction representing an equality constraint.
type Equality struct {
	// expressions will be summed together to form the LHS of ...
	expressions []expression

	// ... a constraint with a certain RHS
	equalTo float64
}

// Initiate a new MILP problem abstraction
func NewProblem() Problem {
	return Problem{}
}

// add a variable and return a reference to that variable
func (p *Problem) AddVariable(coef float64, integer bool) *Variable {

	v := Variable{
		Coefficient: coef,
		Integer:     integer,
	}

	p.variables = append(p.variables, &v)

	return &v
}

// Add an Equality constraint to the problem, given a set of expressions that must equal equalTo.
func (p *Problem) AddEquality(expr []expression, equalTo float64) {
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
func (p *Problem) AddInEquality(expr []expression, smallerThan float64) {
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

// Check whether the expression is legal considering the variables currently present in the problem
func (p *Problem) checkExpression(e expression) bool {

	// check whether the pointer to the variable provided is currently included in the Problem
	for _, v := range p.variables {
		if v == e.variable {
			return true
		}
	}

	return false

}

// Convert the abstract problem representation to its concrete numerical representation.
func (p *Problem) toSolveable() *MILPproblem {
	// TODO: sanity checks before converting

	// get the c vector containing the coefficients of the variables in the objective function
	// simultaneously parse the integrality constraints
	var c []float64
	var integrality []bool
	for _, v := range p.variables {

		// if the Problem is set to be maximized, we assume that all variable coefficients reflect that.
		// To turn this maximization problem into a minimization one, we multiply all coefficients with -1.
		k := v.Coefficient
		if p.maximize {
			k = k * -1
		}

		c = append(c, k)
		integrality = append(integrality, v.Integer)
	}

	// add the equality constraints
	var b []float64
	var Adata []float64
	for _, equality := range p.equalities {

		// build the matrix row for the equality
		equalityRow := make([]float64, len(p.variables))

		for _, exp := range equality.expressions {
			for i, va := range p.variables {
				if exp.variable == va {
					equalityRow[i] = exp.coef
				}
			}
		}

		Adata = append(Adata, equalityRow...)

		// add the RHS of the equality to the b vector
		b = append(b, equality.equalTo)
	}

	// combine the Adata vector into a matrix
	A := mat.NewDense(len(p.equalities), len(p.variables), Adata)

	// get the inequality constraints
	var h []float64
	var Gdata []float64
	for _, inEquality := range p.inequalities {
		inEqualityRow := make([]float64, len(p.variables))

		for _, exp := range inEquality.expressions {
			for i, va := range p.variables {
				if exp.variable == va {
					inEqualityRow[i] = exp.coef
				}
			}
		}

		Gdata = append(Gdata, inEqualityRow...)

		// add the RHS of the equality to the h vector
		h = append(h, inEquality.smallerThan)

	}

	// combine the Gdata vector into a matrix
	var G *mat.Dense
	if len(p.inequalities) > 0 {
		G = mat.NewDense(len(p.inequalities), len(p.variables), Gdata)
	}

	return &MILPproblem{
		c: c,
		A: A,
		b: b,
		G: G,
		h: h,
		integralityConstraints: integrality,
	}
}
