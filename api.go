package ilp

type Problem struct {
	Variables    []*Variable
	Inequalities []Inequality
	Equalities   []Equality
}

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

type Inequality struct {
	// expressions will be summed together to form the LHS of ...
	expressions []expression

	// ... a constraint with a certain RHS
	smallerThan float64
}

type Equality struct {
	// expressions will be summed together to form the LHS of ...
	expressions []expression

	// ... a constraint with a certain RHS
	equalTo float64
}

func NewProblem() Problem {
	return Problem{}
}

// add a variable and return a reference to that variable
func (p *Problem) AddVariable(coef float64, integer bool) *Variable {

	v := Variable{
		Coefficient: coef,
		Integer:     integer,
	}

	p.Variables = append(p.Variables, &v)

	return &v
}

func (p *Problem) AddEquality(expr []expression, equalTo float64) {
	if len(expr) == 0 {
		panic("must add expressions")
	}

	for _, e := range expr {
		if !p.checkExpression(e) {
			panic("provided expression contains a variable that has not been declared to this problem yet")
		}
	}

	p.Equalities = append(p.Equalities, Equality{
		expressions: expr,
		equalTo:     equalTo,
	})

}

func (p *Problem) AddInEquality(expr []expression, smallerThan float64) {
	if len(expr) == 0 {
		panic("must add expressions")
	}

	for _, e := range expr {
		if !p.checkExpression(e) {
			panic("provided expression contains a variable that has not been declared to this problem yet")
		}
	}

	p.Inequalities = append(p.Inequalities, Inequality{
		expressions: expr,
		smallerThan: smallerThan,
	})

}

// Check whether the expression is legal considering the variables currently present in the problem
func (p *Problem) checkExpression(e expression) bool {

	// check whether the pointer to the variable provided is currently included in the Problem
	for _, v := range p.Variables {
		if v == e.variable {
			return true
		}
	}

	return false

}

// //TODO:
// func (p *Problem) toSolveable() *MILPproblem {

// }
