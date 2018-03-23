# GoMILP

<u>*A work in progress.*</u> 

The scope of this project is to build a simple, reliable MILP solver with an easy to use API in pure Go. Several alternatives ([1](https://github.com/draffensperger/golp),[2](https://github.com/lukpank/go-glpk),[3](https://github.com/costela/golpa)) exist in the form of CGO bindings with older LP solver libraries. While excellent pieces of software, I found their dependence on external libraries a big downside for usecases where maximum portability is key.

This project features an implementation of a ('lazy') branch-and-bound method for solving Mixed Integer Linear Programs. The applied branch and bound procedure is basically a heuristic-guided (depth-first **?**) search over an enumeration tree. In this tree, each node is a particular relaxation of the original problem with additional heuristically determined constraints. To solve each LP relaxation, we use [Gonum's excellent implementation]() of the [Simplex](https://en.wikipedia.org/wiki/Simplex_algorithm) algorithm.

**Currently supports the following problem formulation features:**

- [x] integrality constraints
- [x] inequality constraints
- [x] equality constraints
- [x] maximization
- [x] minimization
- [ ] Optional relaxation of nonnegativity constraints (allow decision variables to take negative values)

# Dependencies

Go dependencies are managed using [Go dep](https://github.com/golang/dep). See `Gopkg.lock` and `Gopkg.toml`.

For testing, solutions to randomized MILPs are compared to solutions produced by the GNU Linear Programming Kit, using its [Go bindings](https://github.com/lukpank/go-glpk). To install `libglpk` on macOs, simply run `brew install glpk`.

# TODO list

- [ ] prevent infinite recursion: branch-and-bound (and in particular this implementation) is known to have infinitely recursive edge cases. Make solver cancellable (ideally using the [context](https://golang.org/pkg/context/) API) to exit these cases.
- [ ] Log decisions of branch-and-bound procedure in a tree structure for visualisation and debugging purposes.
- [ ] config API: set number of workers and choice branching heuristic?
- [ ] add more diverse MILP test cases with known solutions.
- [ ] how to deal with matrix degeneracy in subproblems? Currently handled the same way as infeasible subproblems.
- [ ] in branched subproblems: intiate simplex at solution of parent? (using argument of lp.Simplex)
- [ ] does fiddling with the simplex tolerance value improve outcomes?
- [ ] Currently implemented only the simplest branching heuristics. Room for improvement such as expensive branching heuristics like (pseudo-)costs.
- [ ] Enumeration tree exploration heuristics: use priority queue based on heuristics like total path cost or a best-first approach based on earlier solutions.
- [ ] also fun: linear program preprocessing (MATLAB docs: https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html#btv20av)
- [ ] Queue is currently FIFO. For depth-first exploration, we should go with a LIFO queue.
- [ ] Add heuristic determining which node gets explored first (as we are using depth-first search) https://nl.mathworks.com/help/optim/ug/mixed-integer-linear-programming-algorithms.html?s_tid=gn_loc_drop#btzwtmv


- [ ] CI procedure should include race detector and test timeouts
- [ ] sanity checks before converting Problem to a MILPproblem, such as NaN, Inf, and matrix shapes and variable bound domains
- [ ] testing against GLPK is extremely convoluted due to its shitty API. Moreover, its output is sometimes plain wrong (doesnt diagnose unbounded problems).
- [ ] try to formulate more advanced constraints, like sets of values instead of just integrality? Note that having integer sets as constraints is basically the same as having an integrality constraint, and a <= and >= bound. Branching on this type of constraint can be optimized in a neat way (i.e.) ` x>=0, x<=1, x<=0 ~-> x = 0)`
- [ ] dealing with variables that are unrestricted in sign (currently, each var is subject to a nonnegativity constraint)
- [ ] make CLI and Problem serialization format for easy integration with R/python-based analysis tooling for debugging of mathematical properties.
- [ ] write benchmarks for time and space usage
- [ ] small(?) performance gains may be made by switching dense matrix datastructures over to sparse ones for bigger problems. This could be facilitated by employing Gonum's mat.Matrix interface.