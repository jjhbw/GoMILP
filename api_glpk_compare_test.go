package ilp

// import (
// 	"fmt"
// 	"math/rand"
// 	"testing"

// 	"gonum.org/v1/gonum/optimize/convex/lp"

// 	"github.com/lukpank/go-glpk/glpk"
// 	"gonum.org/v1/gonum/floats"
// 	"gonum.org/v1/gonum/mat"
// )

// // Convert the problem to a GLPK problem using its terrible API
// func ToGLPK(p Problem) *glpk.Prob {
// 	converted := glpk.New()

// 	converted.SetProbName("sample")
// 	converted.SetObjName("Z")

// 	if p.maximize {
// 		converted.SetObjDir(glpk.MAX)
// 	} else {
// 		converted.SetObjDir(glpk.MIN)
// 	}

// 	// define the problem dimensions
// 	converted.AddRows(len(p.equalities) + len(p.inequalities))
// 	converted.AddCols(len(p.variables))

// 	// add the variables
// 	for i := 0; i < len(p.variables); i++ {
// 		name := fmt.Sprintf("x%d", i)
// 		colInd := i + 1
// 		converted.SetColName(colInd, name)

// 		// set the objective coeff
// 		converted.SetObjCoef(colInd, p.variables[i].coefficient)

// 		// give all variables a lower bound of 0
// 		converted.SetColBnds(colInd, glpk.LO, 0.0, 0.0)

// 		// set integrality constraint, if any
// 		if p.variables[i].integer {
// 			converted.SetColKind(colInd, glpk.IV)
// 		}
// 	}

// 	// // add the equality constraints
// 	for i, equality := range p.equalities {

// 		// build the matrix row for the equality
// 		equalityCoefs := []float64{0} // add a zero, see details on this weird glpk api nuance below
// 		indices := []int32{0}
// 		for _, exp := range equality.expressions {
// 			for i, va := range p.variables {
// 				if exp.variable == va {
// 					indices = append(indices, int32(i)+1)
// 					equalityCoefs = append(equalityCoefs, exp.coef)
// 				}
// 			}
// 		}

// 		eqRow := converted.AddRows(1)                              // returns the index of the added row
// 		converted.SetRowName(eqRow, fmt.Sprintf("equality_%v", i)) // name the row for debugging purposes
// 		converted.SetMatRow(eqRow, indices, equalityCoefs)         // NOTE: from the docs: "ind[0] and val[0] are ignored", so a leading 0 is given in both vectors."
// 		converted.SetRowBnds(eqRow, glpk.FX, equality.equalTo, 0)
// 	}

// 	// // add the inequality constraints
// 	for i, ineq := range p.inequalities {

// 		// build the matrix row for the equality
// 		inEqualityCoefs := []float64{0} // add a zero, see details on this weird glpk api nuance below
// 		indices := []int32{0}

// 		for _, exp := range ineq.expressions {
// 			for i, va := range p.variables {
// 				if exp.variable == va {
// 					indices = append(indices, int32(i)+1)
// 					inEqualityCoefs = append(inEqualityCoefs, exp.coef)
// 				}
// 			}
// 		}
// 		ineqRow := converted.AddRows(1)                                // returns the index of the added row
// 		converted.SetRowName(ineqRow, fmt.Sprintf("inequality_%v", i)) // name the row for debugging purposes
// 		converted.SetMatRow(ineqRow, indices, inEqualityCoefs)         // NOTE: from the docs: "ind[0] and val[0] are ignored", so a leading 0 is given in both vectors."
// 		converted.SetRowBnds(ineqRow, glpk.FX, ineq.smallerThan, 0)
// 	}

// 	return converted
// }

// // Compare solver outcomes of a specific problem with those of GLPK
// func TestManualCompareWithGLPK(t *testing.T) {
// 	// build an abstract Problem
// 	prob := NewProblem()

// 	// add the variables
// 	v1 := prob.AddVariable("v1").SetCoeff(-1)
// 	v2 := prob.AddVariable("v2").SetCoeff(-2)
// 	v3 := prob.AddVariable("v3").SetCoeff(0).IsInteger()

// 	// add the equality constraints
// 	prob.AddEquality([]expression{
// 		expression{coef: -1, variable: v1},
// 		expression{coef: 2.6, variable: v2},
// 		expression{coef: 1.2, variable: v3},
// 	}, 4)
// 	prob.AddEquality([]expression{
// 		expression{coef: 3, variable: v1},
// 		expression{coef: 1.1, variable: v2},
// 		expression{coef: 1.6, variable: v3},
// 	}, 9)

// 	// set the problem to maximize
// 	prob.Minimize()

// 	// solve the problem using our own code
// 	solution, err := prob.toSolveable().Solve()
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Println(solution.decisionLog)
// 	fmt.Println(solution.solution)

// 	// convert the problem to glpk
// 	glpkProblem := ToGLPK(prob)

// 	// save the problem for debugging purposes
// 	// glpkProblem.WriteLP(nil, "test.lp")

// 	// solve the problem with the integer solver
// 	iocp := glpk.NewIocp()
// 	iocp.SetPresolve(true)
// 	solveError := glpkProblem.Intopt(iocp)
// 	if solveError != nil {
// 		t.Error(solveError)
// 	}

// 	// parse the solutions and compare outcomes
// 	equalSolutions(t, glpkProblem, &solution, &prob, 0.005)

// }

// // // TODO: temporary test for visual debugging
// // func TestTMP_GLPK(t *testing.T) {
// // 	rnd := rand.New(rand.NewSource(155))

// // 	prob := getRandomProblem(-10, 6, 4, rnd)
// // 	milp := prob.toSolveable()

// // 	for i, eq := range prob.equalities {
// // 		fmt.Println("eq", i)
// // 		for _, exp := range eq.expressions {
// // 			fmt.Println(exp)
// // 		}

// // 	}

// // 	fmt.Println("c:")
// // 	fmt.Println(milp.c)
// // 	fmt.Println("integrality:")
// // 	fmt.Println(milp.integralityConstraints)
// // 	fmt.Println("A:")
// // 	fmt.Println(mat.Formatted(milp.A))
// // 	fmt.Println("b:")
// // 	fmt.Println(milp.b)
// // 	fmt.Println("G:")
// // 	fmt.Println(mat.Formatted(milp.G))
// // 	fmt.Println("h:")
// // 	fmt.Println(milp.h)

// // 	// TODO: remove this test
// // 	t.Error()
// // }

// // Compare a bunch of random MILPs with the GLPK output
// func TestAutoCompare(t *testing.T) {
// 	rnd := rand.New(rand.NewSource(155))

// 	testRandomProbCompareWithGLPK(t, 100, 0, 10, rnd)

// }

// // compare random MILPs solved with own solver to GLPK output
// func testRandomProbCompareWithGLPK(t *testing.T, nTest int, pZero float64, maxN int, rnd *rand.Rand) {
// 	// Try a bunch of random LPs
// 	for i := 0; i < nTest; i++ {
// 		n := rnd.Intn(maxN) + 2 // n must be at least two.
// 		m := rnd.Intn(n-1) + 1  // m must be between 1 and n
// 		prob := getRandomProblem(pZero, m, n, rnd)

// 		milp := prob.toSolveable()

// 		// debugging information (in both logs)
// 		t.Log("---Running test number ", i)
// 		fmt.Println("---Test number ", i)

// 		// convert the problem to GLPK
// 		glpkProblem := ToGLPK(prob)
// 		defer glpkProblem.Delete() // we need to manually free up memory of GLPK's CGO implementation

// 		// solve the problem with our own solver
// 		solution, ownErr := milp.Solve()
// 		fmt.Println("own solution:")
// 		fmt.Println(solution.solution.x, solution.solution.z, ownErr)

// 		// Solve GLPK problem with the integer solver
// 		iocp := glpk.NewIocp()
// 		iocp.SetPresolve(true)
// 		GLPKerror := glpkProblem.Intopt(iocp)

// 		// compare errors of both solver outputs
// 		tol := 0.005 // numberical tolerance
// 		if ownErr != nil {
// 			if GLPKerror != nil {
// 				//TODO: compare error messages. If equal: all is well.
// 				if !equalErrors(t, GLPKerror, glpkProblem, ownErr) {
// 					t.Errorf("Problem %v. Errors of both solvers are NOT comparable: GLPKerror = %s vs. own error: %s", i, GLPKerror, ownErr)
// 					// glpkProblem.WriteLP(nil, fmt.Sprintf("problem_%v", i))
// 					summarizeProblem(milp)
// 					t.FailNow()
// 				}
// 			} else {
// 				t.Logf("Problem: %v. Only our own solver returned error: %v", i, ownErr)
// 				t.Log("GLPK solution:")
// 				t.Log("Objective function value:", glpkProblem.MipObjVal())
// 				for a := 0; a < len(milp.c); a++ {
// 					t.Logf("Variable %v value: %g", a, glpkProblem.MipColVal(a+1))
// 				}
// 				// glpkProblem.WriteLP(nil, fmt.Sprintf("problem_%v", i))
// 				summarizeProblem(milp)
// 				t.FailNow()

// 			}
// 		} else {
// 			equalSolutions(t, glpkProblem, &solution, &prob, tol)

// 		}

// 	}
// }

// func summarizeProblem(milp *MILPproblem) {
// 	fmt.Println("Dimensions of own problem:")
// 	fmt.Println("c:")
// 	fmt.Println(milp.c)
// 	fmt.Println("Integrality constraints:")
// 	fmt.Println(milp.integralityConstraints)
// 	fmt.Println("Branching heuristic:")
// 	fmt.Println(milp.branchingHeuristic)
// 	if milp.A != nil {
// 		fmt.Println("A:")
// 		fmt.Println(mat.Formatted(milp.A))
// 	} else {
// 		fmt.Println("A matrix is nil")
// 	}
// 	fmt.Println("b:")
// 	fmt.Println(milp.b)

// 	if milp.G != nil {
// 		fmt.Println("G:")

// 		fmt.Println(mat.Formatted(milp.G))
// 	} else {
// 		fmt.Println("G matrix is nil")
// 	}

// 	fmt.Println("h:")
// 	fmt.Println(milp.h)
// }

// func equalErrors(t *testing.T, glpkError error, glpkProblem *glpk.Prob, ownError error) bool {
// 	// okmsg := "Errors of both solvers are comparable: GLPKerror = %s vs. own error: %s"
// 	glpkStatus := glpkProblem.Status()

// 	// Note that we compare both the error message and the 'problem status'
// 	glpkInfeasible := glpkStatus == glpk.INFEAS || glpkStatus == glpk.NOFEAS || glpkError == glpk.ENOPFS
// 	ownInfeasible := ownError == NO_INTEGER_FEASIBLE_SOLUTION
// 	if glpkInfeasible && ownInfeasible {
// 		// t.Logf(okmsg, glpkError, ownError)
// 		return true
// 	}

// 	if ownError == INITIAL_RELAXATION_NOT_FEASIBLE && glpkError == glpk.ENOPFS {
// 		// t.Logf(okmsg, glpkError, ownError)
// 		return true
// 	}

// 	// Also note that the GLPK integer solver seems to just throw errors around: almost every type of solve failure results in a 'no primal feasible solution'
// 	if ownError == lp.ErrUnbounded && glpkError == glpk.ENOPFS {
// 		return true
// 	}

// 	// t.Logf("Errors of both solvers are NOT comparable: GLPKerror = %s vs. own error: %s", glpkError, ownError)

// 	return false
// }

// func equalSolutions(t *testing.T, glpkProblem *glpk.Prob, solution *MILPsolution, originalProblem *Problem, tolerance float64) bool {
// 	// parse the solutions and compare outcomes
// 	glpkObjectiveVal := glpkProblem.MipObjVal()
// 	if !floats.EqualWithinAbs(solution.solution.z, glpkObjectiveVal, tolerance) {
// 		t.Errorf("Objective function outcome not equal. Own: %g vs. GLPK: %g", solution.solution.z, glpkObjectiveVal)
// 		return false
// 	}
// 	for i := 0; i < len(originalProblem.variables); i++ {

// 		// Check if each solution is equal within a fixed tolerance
// 		if !floats.EqualWithinAbs(solution.solution.x[i], glpkProblem.MipColVal(i+1), tolerance) {
// 			t.Errorf("Decision variable x%v values not equal. Own: %g vs. GLPK: %g", i, solution.solution.x[i], glpkProblem.MipColVal(i+1))
// 			return false
// 		}
// 	}

// 	return true
// }
